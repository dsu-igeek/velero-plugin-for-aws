package ebs_astrolabe

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/vmware-tanzu/astrolabe/pkg/astrolabe"
	"io"
)

type EBSProtectedEntity struct {
	id        astrolabe.ProtectedEntityID
	petm * EBSProtectedEntityTypeManager
}

func NewEBSProtectedEntity(id astrolabe.ProtectedEntityID, petm *EBSProtectedEntityTypeManager) EBSProtectedEntity {
	return EBSProtectedEntity{
		id:   id,
		petm: petm,
	}
}

func (recv EBSProtectedEntity) GetInfo(ctx context.Context) (astrolabe.ProtectedEntityInfo, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) GetCombinedInfo(ctx context.Context) ([]astrolabe.ProtectedEntityInfo, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) describeVolume(volumeID string) (*ec2.Volume, error) {
	req := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{&volumeID},
	}

	res, err := recv.petm.ec2.DescribeVolumes(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if count := len(res.Volumes); count != 1 {
		return nil, errors.Errorf("Expected one volume from DescribeVolumes for volume ID %v, got %v", volumeID, count)
	}

	return res.Volumes[0], nil
}

func getTags(veleroTags map[string]string, volumeTags []*ec2.Tag) []*ec2.Tag {
	var result []*ec2.Tag

	// set Velero-assigned tags
	for k, v := range veleroTags {
		result = append(result, ec2Tag(k, v))
	}

	// copy tags from volume to snapshot
	for _, tag := range volumeTags {
		// we want current Velero-assigned tags to overwrite any older versions
		// of them that may exist due to prior snapshots/restores
		if _, found := veleroTags[*tag.Key]; found {
			continue
		}

		result = append(result, ec2Tag(*tag.Key, *tag.Value))
	}

	return result
}

func ec2Tag(key, val string) *ec2.Tag {
	return &ec2.Tag{Key: &key, Value: &val}
}

func (recv EBSProtectedEntity) Snapshot(ctx context.Context, params map[string]map[string]interface{}) (astrolabe.ProtectedEntitySnapshotID, error) {
	if recv.id.HasSnapshot() {
		return astrolabe.ProtectedEntitySnapshotID{}, errors.Errorf("%s is a snapshot, cannot snapshot a snapshot", recv.id.String())
	}
	volumeID := recv.id.GetBaseID().GetID()
	// describe the volume so we can copy its tags to the snapshot
	volumeInfo, err := recv.describeVolume(volumeID)
	if err != nil {
		return astrolabe.ProtectedEntitySnapshotID{}, err
	}

	tags := make(map[string]string, 0)	// TODO - figure out what we want to do here and how to pass via Astrolabe or if necessary
	snapshotTags := getTags(tags, volumeInfo.Tags)
	snapshotInput := &ec2.CreateSnapshotInput{
		VolumeId: &volumeID,
	}
	if len(snapshotTags) > 0 {
		snapshotInput.TagSpecifications = []*ec2.TagSpecification{
			{
				ResourceType: aws.String(ec2.ResourceTypeSnapshot),
				Tags:         snapshotTags,
			},
		}
	}
	res, err := recv.petm.ec2.CreateSnapshot(snapshotInput)
	if err != nil {
		return astrolabe.ProtectedEntitySnapshotID{}, errors.WithStack(err)
	}

	return astrolabe.NewProtectedEntitySnapshotID(*res.SnapshotId), nil
}

func (recv EBSProtectedEntity) ListSnapshots(ctx context.Context) ([]astrolabe.ProtectedEntitySnapshotID, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) DeleteSnapshot(ctx context.Context, snapshotToDelete astrolabe.ProtectedEntitySnapshotID, params map[string]map[string]interface{}) (bool, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) GetInfoForSnapshot(ctx context.Context, snapshotID astrolabe.ProtectedEntitySnapshotID) (*astrolabe.ProtectedEntityInfo, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) GetComponents(ctx context.Context) ([]astrolabe.ProtectedEntity, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) GetID() astrolabe.ProtectedEntityID {
	panic("implement me")
}

func (recv EBSProtectedEntity) GetDataReader(ctx context.Context) (io.ReadCloser, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) GetMetadataReader(ctx context.Context) (io.ReadCloser, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) Overwrite(ctx context.Context, sourcePE astrolabe.ProtectedEntity, params map[string]map[string]interface{}, overwriteComponents bool) error {
	panic("implement me")
}
