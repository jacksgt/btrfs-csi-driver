package driver

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"k8s.io/klog/v2"
)

const (
	// BtrfsRootPath is the default root path where Btrfs subvolumes will be created
	// This is used for filesystem operations and as a fallback when no subvolumeRoot parameter is provided
	BtrfsRootPath = "/var/lib/btrfs-csi"
	// DefaultQuotaSize is the default quota size if not specified (1GB)
	DefaultQuotaSize = 1073741824 // 1GB in bytes
)

// BtrfsManager handles Btrfs subvolume operations
type BtrfsManager struct {
}

// NewBtrfsManager creates a new BtrfsManager instance
func NewBtrfsManager() *BtrfsManager {
	return &BtrfsManager{}
}

// createBtrfsSubvolume creates a new Btrfs subvolume with quota
func (d *BtrfsDriver) createBtrfsSubvolume(subvolumePath string, sizeBytes int64) error {
	// Ensure the root directory exists
	if err := os.MkdirAll(BtrfsRootPath, 0755); err != nil {
		return fmt.Errorf("failed to create root directory: %v", err)
	}

	// Check if subvolume already exists
	if _, err := os.Stat(subvolumePath); err == nil {
		klog.Infof("Subvolume %s already exists", subvolumePath)
		return nil
	}

	// Create the subvolume
	cmd := exec.Command("chroot", "/host", "btrfs", "subvolume", "create", subvolumePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create btrfs subvolume: %v, output: %s", err, string(output))
	}

	klog.Infof("Created btrfs subvolume: %s", subvolumePath)

	// Set quota if size is specified
	if sizeBytes > 0 {
		if err := d.setSubvolumeQuota(subvolumePath, sizeBytes); err != nil {
			// If quota setting fails, log warning but don't fail the subvolume creation
			klog.Warningf("Failed to set quota for subvolume %s: %v", subvolumePath, err)
			klog.Warningf("Subvolume created without quota - this may lead to unlimited growth")
		}
	}

	return nil
}

// deleteBtrfsSubvolume deletes a Btrfs subvolume
func (d *BtrfsDriver) deleteBtrfsSubvolume(subvolumePath string) error {
	// Check if subvolume exists
	if _, err := os.Stat(subvolumePath); os.IsNotExist(err) {
		klog.Infof("Subvolume %s does not exist, skipping deletion", subvolumePath)
		return nil
	}

	// Delete the subvolume
	cmd := exec.Command("chroot", "/host", "btrfs", "subvolume", "delete", subvolumePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete btrfs subvolume: %v, output: %s", err, string(output))
	}

	klog.Infof("Deleted btrfs subvolume: %s", subvolumePath)
	return nil
}

// setSubvolumeQuota sets a quota for a Btrfs subvolume
func (d *BtrfsDriver) setSubvolumeQuota(subvolumePath string, sizeBytes int64) error {
	// First, check if quotas are enabled
	if !d.areQuotasEnabled() {
		return fmt.Errorf("quotas not enabled")
	}

	// Convert bytes to a more readable format for btrfs
	quotaSize := formatQuotaSize(sizeBytes)

	cmd := exec.Command("chroot", "/host", "btrfs", "qgroup", "limit", quotaSize, subvolumePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set quota: %v, output: %s", err, string(output))
	}

	klog.Infof("Set quota %s for subvolume: %s", quotaSize, subvolumePath)
	return nil
}

// areQuotasEnabled checks if quotas are enabled without trying to enable them
func (d *BtrfsDriver) areQuotasEnabled() bool {
	cmd := exec.Command("chroot", "/host", "btrfs", "qgroup", "show", BtrfsRootPath)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// Exit code 1 means quotas are not enabled
			return false
		}
		// Other errors - assume quotas are not enabled
		return false
	}
	// No error means quotas are enabled
	return true
}

// formatQuotaSize formats the quota size for btrfs command
func formatQuotaSize(sizeBytes int64) string {
	// Convert to human readable format
	if sizeBytes >= 1024*1024*1024 {
		return fmt.Sprintf("%dG", sizeBytes/(1024*1024*1024))
	} else if sizeBytes >= 1024*1024 {
		return fmt.Sprintf("%dM", sizeBytes/(1024*1024))
	} else if sizeBytes >= 1024 {
		return fmt.Sprintf("%dK", sizeBytes/1024)
	}
	return fmt.Sprintf("%dB", sizeBytes)
}

// mountSubvolume mounts a Btrfs subvolume to the target path
func (d *BtrfsDriver) mountSubvolume(subvolumePath, targetPath string) error {
	// Use bind mount to mount the subvolume
	cmd := exec.Command("mount", "--bind", subvolumePath, targetPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to bind mount subvolume: %v, output: %s", err, string(output))
	}

	klog.Infof("Mounted subvolume %s to %s", subvolumePath, targetPath)
	return nil
}

// unmountVolume unmounts a volume from the target path
func (d *BtrfsDriver) unmountVolume(targetPath string) error {
	cmd := exec.Command("umount", targetPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount volume: %v, output: %s", err, string(output))
	}

	klog.Infof("Unmounted volume from %s", targetPath)
	return nil
}

// checkBtrfsSupport checks if Btrfs is supported on the system
func (d *BtrfsDriver) checkBtrfsSupport() error {
	// Check if btrfs command is available
	cmd := exec.Command("chroot", "/host", "btrfs", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("btrfs command not found: %v", err)
	}

	// Check if the root path is on a Btrfs filesystem
	cmd = exec.Command("chroot", "/host", "btrfs", "filesystem", "show", BtrfsRootPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("path %s is not on a Btrfs filesystem: %v, output: %s", BtrfsRootPath, err, string(output))
	}

	klog.Infof("Btrfs support verified for path: %s", BtrfsRootPath)
	return nil
}

// SubvolumeInfo contains information about a Btrfs subvolume
type SubvolumeInfo struct {
	ID   int
	Path string
}

type BtrfsFilesystemUsage struct {
	DeviceSize        int64   // Device size in bytes
	DeviceAllocated   int64   // Device allocated in bytes
	DeviceUnallocated int64   // Device unallocated in bytes
	DeviceMissing     int64   // Device missing in bytes
	DeviceSlack       int64   // Device slack in bytes
	Used              int64   // Used bytes
	FreeEstimated     int64   // Free (estimated) bytes
	FreeEstimatedMin  int64   // Free (estimated) min bytes
	FreeStatfs        int64   // Free (statfs, df) bytes
	DataRatio         float64 // Data ratio
	MetadataRatio     float64 // Metadata ratio
	GlobalReserve     int64   // Global reserve bytes
	GlobalReserveUsed int64   // Global reserve used bytes
	MultipleProfiles  bool    // Multiple profiles (true if "yes", false if "no")
}

// getBtrfsFilesystemUsage parses the output of 'btrfs filesystem usage' for volume usage statistics
func (d *BtrfsDriver) getBtrfsFilesystemUsage(path string) (BtrfsFilesystemUsage, error) {
	usage := BtrfsFilesystemUsage{}

	// Use btrfs filesystem usage to get accurate usage statistics
	cmd := exec.Command("chroot", "/host", "btrfs", "filesystem", "usage", "--raw", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return usage, fmt.Errorf("failed to get btrfs filesystem usage: %v, output: %s", err, string(output))
	}

	// Example output:
	// Overall:
	//     Device size:                       10737418240
	//     Device allocated:                    562036736
	//     Device unallocated:                10175381504
	//     Device missing:                              0
	//     Device slack:                                0
	//     Used:                                   393216
	//     Free (estimated):                  10183770112      (min: 5096079360)
	//     Free (statfs, df):                 10182721536
	//     Data ratio:                               1.00
	//     Metadata ratio:                           2.00
	//     Global reserve:                        5767168      (used: 0)
	//     Multiple profiles:                          no

	// Parse the output to get the usage statistics
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) >= 2 {
			key := strings.TrimSpace(fields[0])
			value := strings.TrimSpace(fields[1])
			if value != "" {
				// required for lines that have trailing output, e.g.
				// `10183770112      (min: 5096079360)`
				value = strings.Fields(fields[1])[0]
			}
			switch key {
			case "Device size":
				usage.DeviceSize, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse device size: %v", err)
				}
			case "Device allocated":
				usage.DeviceAllocated, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse device allocated: %v", err)
				}
			case "Device unallocated":
				usage.DeviceUnallocated, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse device unallocated: %v", err)
				}
			case "Device missing":
				usage.DeviceMissing, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse device missing: %v", err)
				}
			case "Device slack":
				usage.DeviceSlack, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse device slack: %v", err)
				}
			case "Used":
				usage.Used, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse used: %v", err)
				}
			case "Free (estimated)":
				usage.FreeEstimated, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse free estimated: %v", err)
				}
			case "Free (statfs, df)":
				usage.FreeStatfs, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse free statfs: %v", err)
				}
			case "Data ratio":
				usage.DataRatio, err = strconv.ParseFloat(value, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse data ratio: %v", err)
				}
			case "Metadata ratio":
				usage.MetadataRatio, err = strconv.ParseFloat(value, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse metadata ratio: %v", err)
				}
			case "Global reserve":
				usage.GlobalReserve, err = strconv.ParseInt(value, 10, 64)
				if err != nil {
					return usage, fmt.Errorf("failed to parse global reserve: %v", err)
				}
			}
		}
	}
	klog.V(6).Infof("Btrfs filesystem usage of %s: %#v", path, usage)

	return usage, nil
}

// Initialize BtrfsManager in the driver
func (d *BtrfsDriver) initBtrfsManager() error {
	d.btrfsManager = NewBtrfsManager()
	// return d.checkBtrfsSupport()
	return nil
}
