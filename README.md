# Btrfs CSI Driver

A Kubernetes Container Storage Interface (CSI) driver that provides persistent volumes using Btrfs subvolumes with quotas.

## Status

**ALPHA**

## Features

- [x] **Dynamic provisioning of Btrfs Subvolumes**: Each persistent volume is created as a Btrfs subvolume
- [x] **Quota Support**: Automatic quota management for volume size limits
- [x] **Capacity information**: [Storage Capacity](https://kubernetes.io/docs/concepts/storage/storage-capacity/) is exposed to help the scheduler make decisions
- [x] **Container Native**: Full CSI compliance, can be used on Kubernetes or other container orchestrators
- [ ] **Snapshot support**: Kubernetes VolumeSnapshots can be used to create btrfs subvolume snapshot
- [x] **Metrics**: Volume usage information is exposed by the CSI driver (Kubernetes Kubelet exports these as Prometheus metrics)
- [x] **Multiple StorageClasses**: the CSI driver serves multiple StorageClasses which can point to different btrfs filesystems
- [x] **Volume expansion**: allow increasing the size of a volume after creation (online expansion supported)
- [ ] **Volume cloning**: Existing PVCs can be atomically copied to a new PVC by using Btrfs snapshots
- [ ] **Volume specific configuration**: allow dis-/enabling Copy-on-Write (CoW) for individual btrfs subvolumes

## Prerequisites

- Kubernetes cluster with CSI support
- Nodes with at least one Btrfs filesystem
- Nodes must have `btrfs` CLI installed (usually part of `btrfs-progs` package)

## Quick Start

### 1. Build and Deploy

```bash
# Build the driver
make build

# Build Docker image
make docker-build

# Deploy to Kubernetes
make deploy
```

### 2. Verify Installation

```bash
# Check driver status
make status

# Check logs
make logs
```

### 3. Test the Driver

```bash
# Deploy test PVC and Pod
make deploy-test

# Check if the pod is running
kubectl get pods -l app=btrfs-test-pod

# Check the volume
kubectl exec -it btrfs-test-pod -- ls -la /data
```

## Helm Chart

The Helm chart is recommended for deployment of the Btrfs CSI Driver on Kubernetes.
The chart supports extensive configuration through values.
Please refer to the [Helm chart README](./deploy/helm/btrfs-csi/README.md) for detailed configuration options and examples.

```bash
# Install the chart
helm install btrfs-csi ./deploy/helm/btrfs-csi

# Install with custom values
helm install btrfs-csi ./deploy/helm/btrfs-csi -f custom-values.yaml

# Install in a specific namespace
helm install btrfs-csi ./deploy/helm/btrfs-csi --namespace kube-system
```


## Architecture

The driver is deployed as a single `DaemonSet` that is composed of:

- **CSI Provisioner**: Main driver implementing the CSI interface, configured in *distributed provisioning* mode
- **CSI Resizer**: Sidecar container that watches for PVC expansion requests and triggers volume expansion
- **Node Driver Registrar**: Sidecar container for node registration
- **Btrfs Plugin**: Handles Btrfs subvolume creation, deletion, and quota management

## Volume Lifecycle

1. **CreateVolume** (called after creating a PVC): Allocates a new Btrfs subvolume on the target node and prepares it for use.
2. **NodePublishVolume** (called when creating a Pod that uses the PVC): Mounts the subvolume to the target pod's filesystem.
3. **NodeUnpublishVolume** (called when the Pod is deleted): Unmounts the subvolume from the pod.
4. **DeleteVolume** (called when the PVC is deleted): Deletes the Btrfs subvolume from the node, releasing the storage.

### StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: btrfs-local
provisioner: btrfs.csi.k8s.io
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
allowVolumeExpansion: true
```

### PersistentVolumeClaim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: btrfs-local
```

### Volume Expansion

The driver supports **online volume expansion** - volumes can be expanded while they are in use without taking them offline. To expand a volume, simply update the PVC's storage request:

```bash
# Edit the PVC to increase storage
kubectl patch pvc my-pvc -p '{"spec":{"resources":{"requests":{"storage":"2Gi"}}}}'

# Or edit the PVC directly
kubectl edit pvc my-pvc
```

The driver will automatically updates the Btrfs subvolume quota (if enabled) to the new size.
The expanded capacity will be immediately available to the pod.

**Note**: Volume expansion requires the `allowVolumeExpansion: true` setting in the StorageClass.

## Development

### Building

```bash
# Download dependencies
make deps

# Build binary
make build

# Run tests
make test
```

### Testing

```bash
# Deploy test resources
make deploy-test

# Check logs
make logs

# Clean up
make clean-test
```

## Troubleshooting

### Check Btrfs Support

```bash
make check-btrfs
```

### View Driver Logs

```bash
make logs-csi
```

### Check Driver Status

```bash
make status
```

## Security Considerations

The driver runs with `privileged: true` and `SYS_ADMIN` capability, this is required for Btrfs subvolume operations and mount operations.
Consider using Pod Security Standards in production environments.

## License

This project is licensed under the Apache License 2.0.
