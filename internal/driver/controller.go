package driver

import (
	"context"
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// Controller Service Implementation
func (d *BtrfsDriver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.Infof("CreateVolume: called with args %+v", req)

	if err := d.validateCreateVolumeRequest(req); err != nil {
		return nil, err
	}

	volumeID := req.GetName()
	capacity := req.GetCapacityRange().GetRequiredBytes()

	// Determine the target node for this volume
	targetNode := d.nodeID // Default to controller node
	
	// For WaitForFirstConsumer, check accessibility requirements to find the target node
	if req.GetAccessibilityRequirements() != nil {
		for _, topology := range req.GetAccessibilityRequirements().GetPreferred() {
			if hostname, exists := topology.GetSegments()["kubernetes.io/hostname"]; exists {
				targetNode = hostname
				break
			}
		}
		// If no preferred topology found, check requisite
		if targetNode == d.nodeID {
			for _, topology := range req.GetAccessibilityRequirements().GetRequisite() {
				if hostname, exists := topology.GetSegments()["kubernetes.io/hostname"]; exists {
					targetNode = hostname
					break
				}
			}
		}
	}

	klog.Infof("CreateVolume: creating volume %s for node %s", volumeID, targetNode)

	// For WaitForFirstConsumer, we don't create the volume immediately
	// We'll create it in NodePublishVolume when we know the node
	volume := &csi.Volume{
		VolumeId:      volumeID,
		CapacityBytes: capacity,
		VolumeContext: map[string]string{
			"storage.kubernetes.io/csiProvisionerIdentity": "btrfs-csi",
			"targetNode": targetNode,
			"capacity": strconv.FormatInt(capacity, 10),
		},
		ContentSource: req.GetVolumeContentSource(),
	}

	// Add accessibility requirements for local volumes
	volume.AccessibleTopology = []*csi.Topology{
		{
			Segments: map[string]string{
				"kubernetes.io/hostname": targetNode,
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

// Controller validation methods
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