package ebs_astrolabe

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/astrolabe/pkg/astrolabe"
)

type EBSProtectedEntityTypeManager struct {
	logger logrus.FieldLogger
	ec2    *ec2.EC2
	ebs    *ebs.EBS
}

const (
	s3URLKey                 = "s3Url"
	publicURLKey             = "publicUrl"
	kmsKeyIDKey              = "kmsKeyId"
	s3ForcePathStyleKey      = "s3ForcePathStyle"
	bucketKey                = "bucket"
	signatureVersionKey      = "signatureVersion"
	credentialsFileKey       = "credentialsFile"
	credentialProfileKey     = "profile"
	serverSideEncryptionKey  = "serverSideEncryption"
	insecureSkipTLSVerifyKey = "insecureSkipTLSVerify"
	caCertKey                = "caCert"
	regionKey                = "region"
)

const ebsType = "ebs"

// takes AWS session options to create a new session
func getSession(options session.Options) (*session.Session, error) {
	sess, err := session.NewSessionWithOptions(options)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, errors.WithStack(err)
	}
	return sess, nil
}

func NewEBSProtectedEntityTypeManagerVeleroStyle(config map[string]string, log logrus.FieldLogger) (astrolabe.ProtectedEntityTypeManager, error) {

	/*
	if err := veleroplugin.ValidateVolumeSnapshotterConfigKeys(config, regionKey, credentialProfileKey); err != nil {
	return err
	}
	*/
	region := config[regionKey]
	credentialProfile := config[credentialProfileKey]
	if region == "" {
		return nil, errors.Errorf("missing %s in aws configuration", regionKey)
	}

	awsConfig := aws.NewConfig().WithRegion(region)

	sessionOptions := session.Options{Config: *awsConfig, Profile: credentialProfile}
	sess, err := getSession(sessionOptions)
	if err != nil {
		return nil, err
	}

	newPETM := EBSProtectedEntityTypeManager{
		ec2:    ec2.New(sess),
		ebs:    ebs.New(sess),
		logger: log,
	}

	return newPETM, nil
}

func NewEBSProtectedEntityTypeManager(params map[string]interface{}, s3Config astrolabe.S3Config, logger logrus.FieldLogger) (astrolabe.ProtectedEntityTypeManager, error) {

	/*
		if err := veleroplugin.ValidateVolumeSnapshotterConfigKeys(config, regionKey, credentialProfileKey); err != nil {
		return err
		}
	*/
	region := params[regionKey].(string)
	credentialProfile := params[credentialProfileKey].(string)
	if region == "" {
		return nil, errors.Errorf("missing %s in aws configuration", regionKey)
	}

	awsConfig := aws.NewConfig().WithRegion(region)

	sessionOptions := session.Options{Config: *awsConfig, Profile: credentialProfile}
	sess, err := getSession(sessionOptions)
	if err != nil {
		return nil, err
	}

	newPETM := EBSProtectedEntityTypeManager{
		ec2:    ec2.New(sess),
		ebs:    ebs.New(sess),
		logger: logger,
	}

	return newPETM, nil
}

func (recv EBSProtectedEntityTypeManager) GetTypeName() string {
	return "ebs"
}
func (recv EBSProtectedEntityTypeManager) GetProtectedEntity(ctx context.Context, id astrolabe.ProtectedEntityID) (astrolabe.ProtectedEntity, error) {
	return NewEBSProtectedEntity(id, &recv), nil
}

func (recv EBSProtectedEntityTypeManager) GetProtectedEntities(ctx context.Context) ([]astrolabe.ProtectedEntityID, error) {
	more := true
	returnPEIDs := make([]astrolabe.ProtectedEntityID, 0)
	var nextToken *string
	for more {
		var maxResults int64
		maxResults = 1000
		dvi := ec2.DescribeVolumesInput {
			MaxResults: &maxResults,
			NextToken: nextToken,
		}
		dvo, err := recv.ec2.DescribeVolumes(&dvi)
		if err != nil {
			return nil, errors.WithMessage(err, "Failed retrieving using DescribeVolumes")
		}
		nextToken = dvo.NextToken
		if nextToken == nil {
			more = false
		}
		for _, curVolume := range dvo.Volumes {
			peid := astrolabe.NewProtectedEntityID(ebsType, *curVolume.VolumeId)
			returnPEIDs = append(returnPEIDs, peid)
		}
	}
	return returnPEIDs, nil
}

func (recv EBSProtectedEntityTypeManager) Copy(ctx context.Context, pe astrolabe.ProtectedEntity, params map[string]map[string]interface{}, options astrolabe.CopyCreateOptions) (astrolabe.ProtectedEntity, error) {
	panic("implement me")
}

func (recv EBSProtectedEntityTypeManager) CopyFromInfo(ctx context.Context, info astrolabe.ProtectedEntityInfo, params map[string]map[string]interface{}, options astrolabe.CopyCreateOptions) (astrolabe.ProtectedEntity, error) {
	panic("implement me")
}

func (recv EBSProtectedEntityTypeManager) Delete(ctx context.Context, id astrolabe.ProtectedEntityID) error {
	panic("implement me")
}
