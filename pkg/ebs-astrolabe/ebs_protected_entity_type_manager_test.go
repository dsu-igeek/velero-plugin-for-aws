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
}
