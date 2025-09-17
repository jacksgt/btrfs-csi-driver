# Btrfs CSI Driver Deployment Guide

## Overview

This document provides step-by-step instructions for deploying the Btrfs CSI driver to a Kubernetes cluster.

## Prerequisites

### Cluster Requirements
- Kubernetes cluster with CSI support (v1.13+)
- Nodes with Btrfs filesystem support
- `btrfs-progs` installed on all nodes
- Docker or compatible container runtime

### Node Requirements
Each node must have:
- Btrfs filesystem available
- `btrfs-progs` package installed
- Sufficient disk space for subvolumes
- Proper permissions for mount operations

## Deployment Steps

### 1. Prepare the Environment

```bash
# Clone the repository
git clone <repository-url>
cd btrfs-csi

# Build the driver
make build

# Build Docker image
make docker-build
```

### 2. Deploy to Kubernetes

```bash
# Deploy all components
make deploy

# Verify deployment
make status
```

### 3. Verify Installation

```bash
# Check CSI driver status
kubectl get csidriver btrfs.csi.k8s.io

# Check DaemonSet status
kubectl get daemonset -n kube-system btrfs-csi-driver

# Check pod status
kubectl get pods -n kube-system -l app=btrfs-csi-driver

# Check logs
make logs
```

### 4. Test the Driver

```bash
# Deploy test resources
make deploy-test

# Check if test pod is running
kubectl get pods -l app=btrfs-test-pod

# Check the volume
kubectl exec -it btrfs-test-pod -- ls -la /data
```

## Configuration

### StorageClass

The default StorageClass is configured with:
- `volumeBindingMode: WaitForFirstConsumer` - Volumes are created when pods are scheduled
- `reclaimPolicy: Delete` - Volumes are deleted when PVCs are deleted
- `provisioner: btrfs.csi.k8s.io` - Uses our Btrfs CSI driver

### Volume Creation Process

1. **CreateVolume**: Creates volume definition (no actual subvolume yet)
2. **WaitForFirstConsumer**: Waits for pod scheduling
3. **NodePublishVolume**: Creates actual Btrfs subvolume on target node
4. **NodeUnpublishVolume**: Unmounts and deletes subvolume

## Troubleshooting

### Common Issues

#### 1. Btrfs Not Available
```bash
# Check Btrfs support on nodes
make check-btrfs

# Install Btrfs on nodes
# On Ubuntu/Debian:
sudo apt-get install btrfs-progs

# On CentOS/RHEL:
sudo yum install btrfs-progs
```

#### 2. Permission Issues
The driver requires:
- `privileged: true`
- `SYS_ADMIN` capability
- Access to host filesystem

#### 3. Volume Creation Fails
Check logs for:
- Disk space availability
- Btrfs filesystem status
- Mount point permissions

```bash
# Check driver logs
make logs

# Check node status
kubectl describe node <node-name>
```

### Debug Commands

```bash
# View detailed logs
kubectl logs -n kube-system -l app=btrfs-csi-driver -c btrfs-csi-driver --tail=100

# Check CSI driver events
kubectl get events --field-selector involvedObject.name=btrfs.csi.k8s.io

# Check PVC status
kubectl describe pvc <pvc-name>

# Check PV status
kubectl describe pv <pv-name>
```

## Security Considerations

### Required Permissions
- `privileged: true` - Required for mount operations
- `SYS_ADMIN` capability - Required for Btrfs operations
- Host network access - Required for node communication

### Pod Security Standards
In production environments, consider:
- Using Pod Security Standards
- Implementing proper RBAC
- Network policies for CSI communication
- Resource limits and requests

## Monitoring

### Key Metrics to Monitor
- Volume creation/deletion success rate
- Mount/unmount operation duration
- Disk space utilization
- Driver pod health

### Log Levels
- `--v=1` - Basic information
- `--v=2` - Detailed information
- `--v=5` - Debug information (default in manifests)

## Cleanup

### Remove Test Resources
```bash
make clean-test
```

### Uninstall Driver
```bash
make undeploy
```

### Manual Cleanup
```bash
# Remove all resources
kubectl delete -f deploy/kubernetes/storageclass.yaml --ignore-not-found=true
kubectl delete -f deploy/kubernetes/daemonset.yaml --ignore-not-found=true
kubectl delete -f deploy/kubernetes/csi-driver.yaml --ignore-not-found=true
kubectl delete -f deploy/kubernetes/rbac.yaml --ignore-not-found=true
```

## Production Considerations

### High Availability
- Deploy on multiple nodes
- Use anti-affinity rules
- Monitor driver health

### Performance
- Tune Btrfs parameters
- Monitor I/O performance
- Consider SSD storage for better performance

### Backup and Recovery
- Implement volume snapshots
- Regular backup procedures
- Disaster recovery planning

## Support

For issues and questions:
1. Check the logs: `make logs`
2. Verify prerequisites: `make check-btrfs`
3. Review troubleshooting section
4. Check GitHub issues
5. Contact maintainers