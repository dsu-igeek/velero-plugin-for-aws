package ebs_astrolabe

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/astrolabe/pkg/astrolabe"
	"testing"
)

func TestBasic(t *testing.T) {
	params := make(map[string]interface{},0)
	params[regionKey] = "us-west-1"
	params[credentialProfileKey] = ""

	logger := logrus.StandardLogger()
	testMgr, err := NewEBSProtectedEntityTypeManager(params, astrolabe.S3Config{}, logger)
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

	readTestPEID, err := astrolabe.NewProtectedEntityIDFromString("ebs:vol-01483f0d334439471:snap-0b32a457718a4bad4")
	if err != nil {
		t.Fatal(err)
	}
	readTestPE, err := testMgr.GetProtectedEntity(context.Background(), readTestPEID)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("readTestPE = %v", readTestPE)

	testEBSPE := readTestPE.(EBSProtectedEntity)
	buffer := make([] byte, testEBSPE.BlockSize())
	err = testEBSPE.Read(0, 1, buffer)
	if err != nil {
		t.Fatal(err)
	}
}
