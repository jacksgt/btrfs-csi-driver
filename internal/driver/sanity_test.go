//go:build btrfs
// +build btrfs

package driver

// This file runs the CSI sanity test suite against the Btrfs CSI driver implementation.
// To run these tests, use: go test -v ./internal/driver -run TestSanity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-test/pkg/sanity"
)

func TestSanity(t *testing.T) {
	// Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "btrfs-csi-sanity-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	testEndpoint := fmt.Sprintf("unix://%s/csi.sock", tempDir)
	testNodeID := "test-node-123"
	testTargetPath := filepath.Join(tempDir, "target")
	testStagingPath := filepath.Join(tempDir, "staging")

	// Create directories
	if err := os.MkdirAll(testTargetPath, 0755); err != nil {
		t.Fatalf("Failed to create target path: %v", err)
	}
	if err := os.MkdirAll(testStagingPath, 0755); err != nil {
		t.Fatalf("Failed to create staging path: %v", err)
	}

	// Initialize the driver
	driver, err := NewBtrfsDriver(testNodeID, testEndpoint)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	// Start the driver in a goroutine
	go func() {
		if err := driver.Run(); err != nil {
			t.Errorf("Driver failed to run: %v", err)
		}
	}()

	// Wait a moment for the driver to start
	time.Sleep(2 * time.Second)

	// Configure sanity test
	config := &sanity.Config{
		TargetPath:  testTargetPath,
		StagingPath: testStagingPath,
		Address:     testEndpoint,
		SecretsFile: "",
		TestVolumeParameters: map[string]string{
			"subvolumeRoot": filepath.Join(tempDir, "btrfs-root"),
		},
		CreateTargetDir: func(targetPath string) (string, error) {
			return targetPath, nil
		},
		CreateStagingDir: func(stagingPath string) (string, error) {
			return stagingPath, nil
		},
		IDGen: &sanity.DefaultIDGenerator{},
	}

	// Run sanity tests
	sanity.Test(t, config)
}

// TestIdentityService tests the identity service methods
func TestIdentityService(t *testing.T) {
	driver, err := NewBtrfsDriver("test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()

	// Test GetPluginInfo
	pluginInfo, err := driver.GetPluginInfo(ctx, &csi.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo failed: %v", err)
	}
	if pluginInfo.Name != DriverName {
		t.Errorf("Expected driver name %s, got %s", DriverName, pluginInfo.Name)
	}
	if pluginInfo.VendorVersion != Version {
		t.Errorf("Expected version %s, got %s", Version, pluginInfo.VendorVersion)
	}

	// Test GetPluginCapabilities
	capabilities, err := driver.GetPluginCapabilities(ctx, &csi.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("GetPluginCapabilities failed: %v", err)
	}
	if len(capabilities.Capabilities) == 0 {
		t.Error("Expected at least one capability")
	}

	// Test Probe
	probe, err := driver.Probe(ctx, &csi.ProbeRequest{})
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if !probe.Ready.Value {
		t.Error("Expected driver to be ready")
	}
}

// TestControllerService tests the controller service methods
func TestControllerService(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "btrfs-csi-controller-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := NewBtrfsDriver("test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()

	// Test ControllerGetCapabilities
	capabilities, err := driver.ControllerGetCapabilities(ctx, &csi.ControllerGetCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("ControllerGetCapabilities failed: %v", err)
	}
	if len(capabilities.Capabilities) == 0 {
		t.Error("Expected at least one controller capability")
	}

	// Test CreateVolume
	volumeName := "test-volume"
	capacity := int64(1024 * 1024 * 1024) // 1GB

	createReq := &csi.CreateVolumeRequest{
		Name: volumeName,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: capacity,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		Parameters: map[string]string{
			"subvolumeRoot": filepath.Join(tempDir, "btrfs-root"),
		},
	}

	createResp, err := driver.CreateVolume(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}
	if createResp.Volume == nil {
		t.Fatal("Expected volume in response")
	}
	if createResp.Volume.VolumeId == "" {
		t.Error("Expected non-empty volume ID")
	}
	if createResp.Volume.CapacityBytes != capacity {
		t.Errorf("Expected capacity %d, got %d", capacity, createResp.Volume.CapacityBytes)
	}

	// Test ValidateVolumeCapabilities
	validateReq := &csi.ValidateVolumeCapabilitiesRequest{
		VolumeId: createResp.Volume.VolumeId,
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
	}

	validateResp, err := driver.ValidateVolumeCapabilities(ctx, validateReq)
	if err != nil {
		t.Fatalf("ValidateVolumeCapabilities failed: %v", err)
	}
	if validateResp.Confirmed == nil {
		t.Error("Expected confirmed capabilities")
	}

	// Test DeleteVolume
	deleteReq := &csi.DeleteVolumeRequest{
		VolumeId: createResp.Volume.VolumeId,
	}

	_, err = driver.DeleteVolume(ctx, deleteReq)
	if err != nil {
		t.Fatalf("DeleteVolume failed: %v", err)
	}
}

// TestNodeService tests the node service methods
func TestNodeService(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "btrfs-csi-node-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := NewBtrfsDriver("test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()

	// Test NodeGetCapabilities
	capabilities, err := driver.NodeGetCapabilities(ctx, &csi.NodeGetCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("NodeGetCapabilities failed: %v", err)
	}
	if len(capabilities.Capabilities) == 0 {
		t.Error("Expected at least one node capability")
	}

	// Test NodeGetInfo
	nodeInfo, err := driver.NodeGetInfo(ctx, &csi.NodeGetInfoRequest{})
	if err != nil {
		t.Fatalf("NodeGetInfo failed: %v", err)
	}
	if nodeInfo.NodeId != "test-node" {
		t.Errorf("Expected node ID 'test-node', got %s", nodeInfo.NodeId)
	}
	if nodeInfo.AccessibleTopology == nil {
		t.Error("Expected accessible topology")
	}
}

// TestVolumeExpansion tests volume expansion functionality
func TestVolumeExpansion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "btrfs-csi-expansion-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := NewBtrfsDriver("test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()

	// Create a test volume first
	volumeName := "expansion-test-volume"
	initialCapacity := int64(1024 * 1024 * 1024) // 1GB

	createReq := &csi.CreateVolumeRequest{
		Name: volumeName,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: initialCapacity,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		Parameters: map[string]string{
			"subvolumeRoot": filepath.Join(tempDir, "btrfs-root"),
		},
	}

	createResp, err := driver.CreateVolume(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	// Test volume expansion
	newCapacity := int64(2 * 1024 * 1024 * 1024) // 2GB
	expandReq := &csi.ControllerExpandVolumeRequest{
		VolumeId: createResp.Volume.VolumeId,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: newCapacity,
		},
	}

	expandResp, err := driver.ControllerExpandVolume(ctx, expandReq)
	if err != nil {
		t.Fatalf("ControllerExpandVolume failed: %v", err)
	}
	if expandResp.CapacityBytes != newCapacity {
		t.Errorf("Expected expanded capacity %d, got %d", newCapacity, expandResp.CapacityBytes)
	}
	if !expandResp.NodeExpansionRequired {
		t.Error("Expected node expansion to be required")
	}

	// Clean up
	deleteReq := &csi.DeleteVolumeRequest{
		VolumeId: createResp.Volume.VolumeId,
	}
	_, err = driver.DeleteVolume(ctx, deleteReq)
	if err != nil {
		t.Fatalf("DeleteVolume failed: %v", err)
	}
}

// Benchmark tests for performance testing
func BenchmarkCreateVolume(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "btrfs-csi-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := NewBtrfsDriver("test-node", "unix:///tmp/test.sock")
	if err != nil {
		b.Fatalf("Failed to create driver: %v", err)
	}

	ctx := context.Background()
	capacity := int64(1024 * 1024 * 1024) // 1GB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createReq := &csi.CreateVolumeRequest{
			Name: fmt.Sprintf("bench-volume-%d", i),
			CapacityRange: &csi.CapacityRange{
				RequiredBytes: capacity,
			},
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			Parameters: map[string]string{
				"subvolumeRoot": filepath.Join(tempDir, "btrfs-root"),
			},
		}

		createResp, err := driver.CreateVolume(ctx, createReq)
		if err != nil {
			b.Fatalf("CreateVolume failed: %v", err)
		}

		// Clean up immediately
		deleteReq := &csi.DeleteVolumeRequest{
			VolumeId: createResp.Volume.VolumeId,
		}
		_, err = driver.DeleteVolume(ctx, deleteReq)
		if err != nil {
			b.Fatalf("DeleteVolume failed: %v", err)
		}
	}
}
