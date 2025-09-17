package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/drivers/pkg/csi-common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/klog/v2"
)

const (
	DriverName = "btrfs.csi.k8s.io"
	Version    = "0.0.1"
)

type BtrfsDriver struct {
	*csicommon.CSIDriver
	nodeID        string
	kubeconfig    string
	btrfsManager  *BtrfsManager
}

func NewBtrfsDriver(nodeID, endpoint, kubeconfig string) (*BtrfsDriver, error) {
	klog.Infof("Driver: %v version: %v", DriverName, Version)
	
	if kubeconfig != "" {
		klog.Infof("Using kubeconfig file: %s", kubeconfig)
	} else {
		klog.Infof("Using in-cluster Kubernetes configuration")
	}

	csiDriver := csicommon.NewCSIDriver(DriverName, Version, nodeID)
	if csiDriver == nil {
		return nil, fmt.Errorf("failed to initialize CSI Driver")
	}

	csiDriver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_GET_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
	})

	csiDriver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
	})

	btrfsDriver := &BtrfsDriver{
		CSIDriver:  csiDriver,
		nodeID:     nodeID,
		kubeconfig: kubeconfig,
	}

	// Initialize Btrfs manager
	if err := btrfsDriver.initBtrfsManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize Btrfs manager: %v", err)
	}

	// Create GRPC servers
	btrfsDriver.CSIDriver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_GET_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
	})


	return btrfsDriver, nil
}

func (d *BtrfsDriver) Run() error {
	s := csicommon.NewNonBlockingGRPCServer()
	s.Start("unix://tmp/csi.sock", d, d, d)
	s.Wait()
	return nil
}

// Controller Service Implementation
func (d *BtrfsDriver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.Infof("CreateVolume: called with args %+v", req)

	if err := d.validateCreateVolumeRequest(req); err != nil {
		return nil, err
	}

	volumeID := req.GetName()
	capacity := req.GetCapacityRange().GetRequiredBytes()

	// For WaitForFirstConsumer, we don't create the volume immediately
	// We'll create it in NodePublishVolume when we know the node
	volume := &csi.Volume{
		VolumeId:      volumeID,
		CapacityBytes: capacity,
		VolumeContext: map[string]string{
			"storage.kubernetes.io/csiProvisionerIdentity": "btrfs-csi",
		},
		ContentSource: req.GetVolumeContentSource(),
	}

	// Add accessibility requirements for local volumes
	volume.AccessibleTopology = []*csi.Topology{
		{
			Segments: map[string]string{
				"kubernetes.io/hostname": d.nodeID,
			},
		},
	}

	return &csi.CreateVolumeResponse{
		Volume: volume,
	}, nil
}

func (d *BtrfsDriver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.Infof("DeleteVolume: called with args %+v", req)

	if err := d.validateDeleteVolumeRequest(req); err != nil {
		return nil, err
	}

	volumeID := req.GetVolumeId()
	
	// For local volumes, we need to delete the subvolume on the specific node
	// This will be handled by the node where the volume was created
	klog.Infof("Volume %s will be deleted when NodeUnpublishVolume is called", volumeID)

	return &csi.DeleteVolumeResponse{}, nil
}

func (d *BtrfsDriver) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	klog.Infof("ValidateVolumeCapabilities: called with args %+v", req)

	if err := d.validateValidateVolumeCapabilitiesRequest(req); err != nil {
		return nil, err
	}

	// For now, we support all requested capabilities
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
		},
	}, nil
}

func (d *BtrfsDriver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.Infof("ControllerGetCapabilities: called with args %+v", req)

	capabilities := []*csi.ControllerServiceCapability{
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
				},
			},
		},
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_GET_VOLUME,
				},
			},
		},
		{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
				},
			},
		},
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: capabilities,
	}, nil
}

func (d *BtrfsDriver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *BtrfsDriver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}


// Node Service Implementation
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

	// Create Btrfs subvolume for this volume
	subvolumePath := filepath.Join("/var/lib/btrfs-csi", volumeID)
	// For now, use a default size since NodePublishVolume doesn't have capacity info
	// In a real implementation, you might want to store this info during CreateVolume
	if err := d.createBtrfsSubvolume(subvolumePath, DefaultQuotaSize); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create btrfs subvolume: %v", err)
	}

	// Mount the subvolume to target path
	if err := d.mountSubvolume(subvolumePath, targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mount subvolume: %v", err)
	}

	klog.Infof("NodePublishVolume: volume %s published at %s", volumeID, targetPath)

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

	// Remove target directory
	if err := os.RemoveAll(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove target directory %s: %v", targetPath, err)
	}

	// Delete the Btrfs subvolume
	subvolumePath := filepath.Join("/var/lib/btrfs-csi", volumeID)
	if err := d.deleteBtrfsSubvolume(subvolumePath); err != nil {
		klog.Warningf("Failed to delete btrfs subvolume %s: %v", subvolumePath, err)
	}

	klog.Infof("NodeUnpublishVolume: volume %s unpublished from %s", volumeID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *BtrfsDriver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
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

// Validation methods
func (d *BtrfsDriver) validateCreateVolumeRequest(req *csi.CreateVolumeRequest) error {
	if req.GetName() == "" {
		return status.Error(codes.InvalidArgument, "volume name is required")
	}

	if req.GetCapacityRange() == nil {
		return status.Error(codes.InvalidArgument, "capacity range is required")
	}

	return nil
}

func (d *BtrfsDriver) validateDeleteVolumeRequest(req *csi.DeleteVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "volume ID is required")
	}

	return nil
}

func (d *BtrfsDriver) validateValidateVolumeCapabilitiesRequest(req *csi.ValidateVolumeCapabilitiesRequest) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "volume ID is required")
	}

	if req.GetVolumeCapabilities() == nil {
		return status.Error(codes.InvalidArgument, "volume capabilities is required")
	}

	return nil
}

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

// Identity Service Implementation
func (d *BtrfsDriver) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	klog.Infof("GetPluginInfo: called with args %+v", req)

	return &csi.GetPluginInfoResponse{
		Name:          DriverName,
		VendorVersion: Version,
	}, nil
}

func (d *BtrfsDriver) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	klog.Infof("GetPluginCapabilities: called with args %+v", req)

	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

func (d *BtrfsDriver) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	klog.Infof("Probe: called with args %+v", req)

	return &csi.ProbeResponse{
		Ready: wrapperspb.Bool(true),
	}, nil
}