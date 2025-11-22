package driver

import (
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"k8s.io/klog/v2"
)

const (
	DriverName = "btrfs.csi.k8s.io"
	// TODO: dynamically set this during release
	Version    = "0.0.1"
)

type BtrfsDriver struct {
	*csicommon.CSIDriver
	nodeID       string
	endpoint     string
	btrfsManager *BtrfsManager
}

func NewBtrfsDriver(nodeID, endpoint string) (*BtrfsDriver, error) {
	klog.Infof("Driver: %v version: %v", DriverName, Version)

	csiDriver := csicommon.NewCSIDriver(DriverName, Version, nodeID)
	if csiDriver == nil {
		return nil, fmt.Errorf("failed to initialize CSI Driver")
	}

	csiDriver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
	})

	btrfsDriver := &BtrfsDriver{
		CSIDriver: csiDriver,
		nodeID:    nodeID,
		endpoint:  endpoint,
	}

	// Advertise controller capabilities
	btrfsDriver.CSIDriver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_GET_VOLUME,
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	})
	klog.Infof("Initialized as controller service")

	// Initialize node service
	if err := btrfsDriver.initBtrfsManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize Btrfs manager: %v", err)
	}
	klog.Infof("Initialized as node service with Btrfs support")

	return btrfsDriver, nil
}

func (d *BtrfsDriver) Run() error {
	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(d.endpoint, d, d, d)
	s.Wait()
	return nil
}
