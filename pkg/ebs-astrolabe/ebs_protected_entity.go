package ebs_astrolabe

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/vmware-tanzu/astrolabe/pkg/astrolabe"
	"github.com/vmware-tanzu/astrolabe/pkg/util"
	"io"
	"sync"
)

type EBSProtectedEntity struct {
	id                  astrolabe.ProtectedEntityID
	petm                *EBSProtectedEntityTypeManager
	blockSize           *int
	blockInfoCache      *map[int64]ebs.Block
	blockInfoCacheMutex *sync.Mutex
}

func NewEBSProtectedEntity(id astrolabe.ProtectedEntityID, petm *EBSProtectedEntityTypeManager) EBSProtectedEntity {
	var blockInfoCacheMutex sync.Mutex
	blockInfoCache := make(map[int64]ebs.Block, 0)
	blockSize := -1
	return EBSProtectedEntity{
		id:                  id,
		petm:                petm,
		blockSize:           &blockSize,
		blockInfoCache:      &blockInfoCache,
		blockInfoCacheMutex: &blockInfoCacheMutex,
	}
}

func (recv EBSProtectedEntity) GetInfo(ctx context.Context) (astrolabe.ProtectedEntityInfo, error) {
	if recv.id.HasSnapshot() {
		// TODO - fix GetInfoForSnapshot API to not return a pointer to ProtectedEntityInfo
		snapshotInfo, err := recv.GetInfoForSnapshot(ctx, recv.id.GetSnapshotID())
		// TODO - check for err and don't dereference if err
		return *snapshotInfo, err
	} else {
		dvi := ec2.DescribeVolumesInput{
			VolumeIds: []*string{aws.String(recv.id.GetID())},
		}
		dvo, err := recv.petm.ec2.DescribeVolumes(&dvi)
		if err != nil {
			return nil, errors.WithMessagef(err, "DescribeVolumes failed for EBS Protected Entity %s", recv.id.String())
		}
		name := ""
		for _, checkTag := range dvo.Volumes[0].Tags {
			if *checkTag.Key == "Name" {
				name = *checkTag.Value
			}
		}
		return astrolabe.NewProtectedEntityInfo(recv.id,
			name,
			(*dvo.Volumes[0].Size) * 1024 * 1024 * 1024,	// Convert GiB -> bytes
			nil,
			nil,
			nil,
			nil), nil
	}
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

	tags := make(map[string]string, 0) // TODO - figure out what we want to do here and how to pass via Astrolabe or if necessary
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
	dsi := &ec2.DescribeSnapshotsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("volume-id"),
				Values: []*string{
					aws.String(recv.id.GetID()),
				},
			},
		},
	}

	result, err := recv.petm.ec2.DescribeSnapshots(dsi)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to list snapshots for %s", recv.id.String())
	}
	returnSnapshotIDs := make([]astrolabe.ProtectedEntitySnapshotID, len(result.Snapshots))
	for curSnapshotNum, curSnapshot := range result.Snapshots {
		returnSnapshotIDs[curSnapshotNum] = astrolabe.NewProtectedEntitySnapshotID(*curSnapshot.SnapshotId)
	}
	return returnSnapshotIDs, nil
}

func (recv EBSProtectedEntity) DeleteSnapshot(ctx context.Context, snapshotToDelete astrolabe.ProtectedEntitySnapshotID, params map[string]map[string]interface{}) (bool, error) {
	panic("implement me")
}

func (recv EBSProtectedEntity) GetInfoForSnapshot(ctx context.Context, snapshotID astrolabe.ProtectedEntitySnapshotID) (*astrolabe.ProtectedEntityInfo, error) {
	dsi := ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{aws.String(snapshotID.String())},
	}
	dvo, err := recv.petm.ec2.DescribeSnapshots(&dsi)
	if err != nil {
		return nil, errors.WithMessagef(err, "DescribeSnapshots failed for EBS Protected Entity %s", recv.id.String())
	}
	pei  := astrolabe.NewProtectedEntityInfo(recv.id.IDWithSnapshot(snapshotID),
		*dvo.Snapshots[0].Description,
		(*dvo.Snapshots[0].VolumeSize) * 1024 * 1024 * 1024,	// Convert GiB -> bytes
		nil,
		nil,
		nil,
		nil)
	return &pei, nil
}

func (recv EBSProtectedEntity) GetComponents(ctx context.Context) ([]astrolabe.ProtectedEntity, error) {
	return make([]astrolabe.ProtectedEntity, 0), nil
}

func (recv EBSProtectedEntity) GetID() astrolabe.ProtectedEntityID {
	return recv.id
}

func (recv EBSProtectedEntity) GetDataReader(ctx context.Context) (io.ReadCloser, error) {
	return util.NewBlockSourceReader(recv, recv.petm.logger), nil
}

func (recv EBSProtectedEntity) GetMetadataReader(ctx context.Context) (io.ReadCloser, error) {
	return nil, nil
}

func (recv EBSProtectedEntity) Overwrite(ctx context.Context, sourcePE astrolabe.ProtectedEntity, params map[string]map[string]interface{}, overwriteComponents bool) error {
	panic("implement me")
}

func (recv EBSProtectedEntity) Read(startBlock uint64, numBlocks uint64, buffer []byte) (uint64, error) {
	if !recv.id.HasSnapshot() {
		return 0, errors.Errorf("EBSProtectedEntity %s is not a snapshot", recv.id.String())
	}
	total := uint64(0)
	for curBlock:= startBlock; curBlock < startBlock + numBlocks; curBlock++ {
		curBlockInt64 := int64(curBlock)
		blockToken, err := recv.getBlockTokenForIndex(curBlockInt64)
		if err != nil {
			return 0, err
		}
		gsbi := ebs.GetSnapshotBlockInput{
			BlockIndex: &curBlockInt64,
			BlockToken: blockToken,
			SnapshotId: aws.String(recv.id.GetSnapshotID().String()),
		}
		gsbo, err := recv.petm.ebs.GetSnapshotBlock(&gsbi)
		blockOffset := int(curBlock - startBlock)
		bufOffset := blockOffset * *recv.blockSize
		bytesRead, err := io.ReadFull(gsbo.BlockData, buffer[bufOffset:bufOffset + *recv.blockSize])
		total = total + uint64(bytesRead)
		if bytesRead != *recv.blockSize {
			return total, errors.Errorf("Expected %d bytes, got %d at block #", *recv.blockSize, bytesRead, curBlockInt64)
		}
		if err != nil {
			return total, errors.WithMessagef(err,"Failed at block %d", curBlockInt64 )
		}
	}
	return total, nil
}

func (recv EBSProtectedEntity) BlockSize() int {
	recv.blockInfoCacheMutex.Lock()
	defer recv.blockInfoCacheMutex.Unlock()
	if *recv.blockSize < 0 {
		recv.loadBlockInfoStartingAt(0)
	}
	return *recv.blockSize
}

func (recv EBSProtectedEntity) loadBlockInfoStartingAt(blockIndex int64) error {
	maxResults := int64(1000)
	lsbi := ebs.ListSnapshotBlocksInput{
		MaxResults:         &maxResults,
		NextToken:          nil,
		SnapshotId:         aws.String(recv.id.GetSnapshotID().String()),
		StartingBlockIndex: &blockIndex,
	}
	lsbo, err := recv.petm.ebs.ListSnapshotBlocks(&lsbi)
	if err != nil {
		return errors.WithMessagef(err, "Could not get block token for index %d for EBS Protected Entity %s", blockIndex, recv.id.String())
	}
	*recv.blockSize = int(*lsbo.BlockSize)
	for _, curBlock := range lsbo.Blocks {
		(*recv.blockInfoCache)[*curBlock.BlockIndex] = *curBlock
	}
	return nil
}
func (recv EBSProtectedEntity) getBlockTokenForIndex(blockIndex int64) (*string, error) {
	if !recv.id.HasSnapshot() {
		return nil, errors.Errorf("EBS Protected Entity %s is not a snapshot", recv.id.String())
	}
	recv.blockInfoCacheMutex.Lock()
	defer recv.blockInfoCacheMutex.Unlock()
	block, ok := (*recv.blockInfoCache)[blockIndex]
	if !ok {
		err := recv.loadBlockInfoStartingAt(blockIndex)
		if err != nil {
			return nil, err
		}
		block, ok = (*recv.blockInfoCache)[blockIndex]
		if !ok {
			return nil, errors.Errorf("block info not available for block %d in EBS Protected Entity %s",
				blockIndex, recv.id.String())
		}
	}
	return block.BlockToken, nil
}

func (recv EBSProtectedEntity) Capacity() int64 {
	peInfo, err := recv.GetInfo(context.TODO())
	if err != nil {
		return 0
	}
	return peInfo.GetSize()
}
func (recv EBSProtectedEntity) Close() error {
	recv.blockInfoCacheMutex.Lock()
	defer recv.blockInfoCacheMutex.Unlock()
	// Clear the block info cache on "closing" and don't leave it hanging around (could be big)
	blockInfoCache := make(map[int64]ebs.Block, 0)
	recv.blockInfoCache = &blockInfoCache
	*recv.blockSize = -1
	return nil
}
