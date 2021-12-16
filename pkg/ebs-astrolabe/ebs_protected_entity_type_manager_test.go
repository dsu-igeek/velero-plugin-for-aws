package ebs_astrolabe

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/astrolabe/pkg/astrolabe"
	"io"
	"net"
	"testing"
)

func TestBasic(t *testing.T) {
	params := make(map[string]interface{},0)
	params[regionKey] = "us-west-1"
	params[credentialProfileKey] = ""

	logger := logrus.StandardLogger()
	testMgr, err := NewEBSProtectedEntityTypeManager(params, astrolabe.S3Config{
		Port:      9000,
		Host:      net.IPv4(127,0,0,1),
		AccessKey: "notanaccesskey",
		Secret:    "notasecret",
		Region:    "astrolabe",
		UseHttp:   true,
	}, logger)
	if err != nil {
		t.Fatal(err)
	}

	ebsPEs, err := testMgr.GetProtectedEntities(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for curPENum, curEBSPE := range ebsPEs {
		fmt.Printf("%d: %s\n", curPENum, curEBSPE.String())
	}

	readTestPEID, err := astrolabe.NewProtectedEntityIDFromString("ebs:vol-06a67cc9ebac43807:snap-0b3a6e1d600effc8c")
	if err != nil {
		t.Fatal(err)
	}
	readTestPE, err := testMgr.GetProtectedEntity(context.Background(), readTestPEID)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("readTestPE = %v", readTestPE)

	testEBSPE := readTestPE.(EBSProtectedEntity)
	buffer := make([] byte, 32*1024)
	dr, err := testEBSPE.GetDataReader(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	info, err := readTestPE.GetInfo(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	bytesToRead := info.GetSize()
	totalBytesRead := int64(0)
	for true {
		bytesRead, err := io.ReadFull(dr, buffer)
		totalBytesRead += int64(bytesRead)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if totalBytesRead > bytesToRead {
			t.Fatalf("Read too many bytes - expected %d, got %d", bytesToRead, totalBytesRead)
		}
	}
	if totalBytesRead != bytesToRead {
		t.Fatalf("Didn't get expected number of bytes %d, got %d", bytesToRead, totalBytesRead)
	}
}
