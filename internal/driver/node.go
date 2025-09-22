package driver

import (
	"context"
	"os"
	"path/filepath"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// Node Service Implementation
// TODO: I think this is not needed, we should just mount the subvolume to the pod dir in NodePublishVolume.
// Do we set NODE_STAGE_VOLUME capability?
func (d *BtrfsDriver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.Infof("NodeStageVolume: called with args %+v", req)

	if err := d.validateNodeStageVolumeRequest(req); err != nil {
		return nil, err
	}

	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()

	// Create staging directory
	if err := os.MkdirAll(stagingTargetPath, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create staging directory %s: %v", stagingTargetPath, err)
	}

	// For local volumes, we don't need to mount anything at staging
	// The actual volume will be mounted at publish time
	klog.Infof("NodeStageVolume: volume %s staged at %s", volumeID, stagingTargetPath)

	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *BtrfsDriver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.Infof("NodeUnstageVolume: called with args %+v", req)

	if err := d.validateNodeUnstageVolumeRequest(req); err != nil {
		return nil, err
	}

	stagingTargetPath := req.GetStagingTargetPath()

	// Remove staging directory
	if err := os.RemoveAll(stagingTargetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove staging directory %s: %v", stagingTargetPath, err)
	}

	klog.Infof("NodeUnstageVolume: volume unstaged from %s", stagingTargetPath)

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *BtrfsDriver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	klog.Infof("NodePublishVolume: called with args %+v", req)

	if err := d.validateNodePublishVolumeRequest(req); err != nil {
		return nil, err
	}

	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create target directory %s: %v", targetPath, err)
	}

	// Get subvolume root from volume context
	subvolumeRoot := d.getSubvolumeRootFromVolumeContext(req.GetVolumeContext())

	// The subvolume should already exist (created in CreateVolume)
	subvolumePath := filepath.Join(subvolumeRoot, volumeID)

	// Check if subvolume exists
	if _, err := os.Stat(subvolumePath); os.IsNotExist(err) {
		return nil, status.Errorf(codes.NotFound, "subvolume %s does not exist", subvolumePath)
	}

	// Mount the existing subvolume to target path
	if err := d.mountSubvolume(subvolumePath, targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mount subvolume: %v", err)
	}

	klog.Infof("NodePublishVolume: volume %s mounted at %s", volumeID, targetPath)

	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *BtrfsDriver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.Infof("NodeUnpublishVolume: called with args %+v", req)

	if err := d.validateNodeUnpublishVolumeRequest(req); err != nil {
		return nil, err
	}

	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	// Unmount the volume
	if err := d.unmountVolume(targetPath); err != nil {
		klog.Warningf("Failed to unmount volume at %s: %v", targetPath, err)
	}

	klog.Infof("NodeUnpublishVolume: volume %s removed from %s", volumeID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *BtrfsDriver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	klog.Infof("NodeGetVolumeStats: called with args %+v", req)

	// Validate the request
	if req.GetVolumeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	volumePath := req.GetVolumePath()
	if volumePath == "" {
		return nil, status.Error(codes.InvalidArgument, "volume path is required")
	}

	// Get volume statistics using btrfs commands
	usage, err := d.getBtrfsFilesystemUsage(volumePath)
	if err != nil {
		klog.Errorf("Failed to get volume stats for %s: %v", volumePath, err)
		return nil, status.Errorf(codes.Internal, "failed to get volume stats: %v", err)
	}

	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			{
				Unit:      csi.VolumeUsage_BYTES,
				Available: usage.FreeEstimated,
				Total:     usage.DeviceSize,
				Used:      usage.Used,
			},
		},
	}, nil
}

func (d *BtrfsDriver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.Infof("NodeGetCapabilities: called with args %+v", req)

	capabilities := []*csi.NodeServiceCapability{
		{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
				},
			},
		},
		{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
				},
			},
		},
	}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: capabilities,
	}, nil
}

func (d *BtrfsDriver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.Infof("NodeGetInfo: called with args %+v", req)

	return &csi.NodeGetInfoResponse{
		NodeId: d.nodeID,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				"kubernetes.io/hostname": d.nodeID,
			},
		},
	}, nil
}

// Node validation methods
func (d *BtrfsDriver) validateNodeStageVolumeRequest(req *csi.NodeStageVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "volume ID is required")
	}

	if req.GetStagingTargetPath() == "" {
		return status.Error(codes.InvalidArgument, "staging target path is required")
	}

	return nil
}

func (d *BtrfsDriver) validateNodeUnstageVolumeRequest(req *csi.NodeUnstageVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "volume ID is required")
	}

	if req.GetStagingTargetPath() == "" {
		return status.Error(codes.InvalidArgument, "staging target path is required")
	}

	return nil
}

func (d *BtrfsDriver) validateNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "volume ID is required")
	}

	if req.GetStagingTargetPath() == "" {
		return status.Error(codes.InvalidArgument, "staging target path is required")
	}

	if req.GetTargetPath() == "" {
		return status.Error(codes.InvalidArgument, "target path is required")
	}

	return nil
}

func (d *BtrfsDriver) validateNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "volume ID is required")
	}

	if req.GetTargetPath() == "" {
		return status.Error(codes.InvalidArgument, "target path is required")
	}

	return nil
}
