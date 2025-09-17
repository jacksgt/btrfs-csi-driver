package main

import (
	"flag"
	"os"

	"github.com/btrfs-csi/driver/internal/driver"
	"k8s.io/klog/v2"
)

var (
	endpoint = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeID   = flag.String("nodeid", "", "node id")
)

func main() {
	// Parse our custom flags first
	flag.Parse()

	// Initialize klog with a separate flag set to avoid conflicts
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	klogFlags.Set("logtostderr", "true")

	if *nodeID == "" {
		klog.Fatalf("nodeid is required")
	}

	drv, err := driver.NewBtrfsDriver(*nodeID, *endpoint)
	if err != nil {
		klog.Fatalf("Failed to initialize driver: %v", err)
	}

	klog.Infof("Starting Btrfs CSI driver on node %s", *nodeID)
	if err := drv.Run(); err != nil {
		klog.Fatalf("Failed to run driver: %v", err)
	}

	os.Exit(0)
}
