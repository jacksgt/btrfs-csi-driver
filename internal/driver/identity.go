package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/klog/v2"
)

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
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS,
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
