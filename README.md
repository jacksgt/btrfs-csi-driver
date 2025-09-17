# Btrfs CSI Driver

A Kubernetes Container Storage Interface (CSI) driver that provides persistent volumes using Btrfs subvolumes with quotas.

## Features

- **Btrfs Subvolumes**: Each persistent volume is backed by a Btrfs subvolume
- **Quota Support**: Automatic quota enforcement for each volume
- **Local Volumes**: Uses Kubernetes local volume type for node-specific storage
- **WaitForFirstConsumer**: Volumes are created only when a pod is scheduled
- **Node Driver Registrar**: Automatic node registration and health monitoring
- **Flexible Authentication**: Supports both in-cluster and kubeconfig-based authentication

## Prerequisites

- Kubernetes cluster with CSI support
- Nodes with Btrfs filesystem support
- `btrfs-progs` installed on all nodes
- Docker or compatible container runtime

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

## Architecture

The driver consists of:

- **CSI Driver**: Main driver implementing the CSI interface
- **Btrfs Manager**: Handles Btrfs subvolume creation, deletion, and quota management
- **Node Driver Registrar**: Sidecar container for node registration

## Volume Lifecycle

1. **CreateVolume**: Creates a volume definition (no actual subvolume yet)
2. **WaitForFirstConsumer**: Waits for a pod to be scheduled
3. **NodePublishVolume**: Creates the actual Btrfs subvolume on the target node
4. **NodeUnpublishVolume**: Unmounts and deletes the subvolume

## Configuration

### Command Line Options

The driver supports the following command-line flags:

- `--endpoint`: CSI endpoint (default: `unix://tmp/csi.sock`)
- `--nodeid`: Node ID (required)
- `--kubeconfig`: Path to kubeconfig file (optional, defaults to in-cluster config)
- `--v`: Log level (default: 0)

Example usage:
```bash
# Using in-cluster configuration (default in Kubernetes)
./btrfs-csi --nodeid=node-1 --endpoint=unix://tmp/csi.sock

# Using local kubeconfig file
./btrfs-csi --nodeid=node-1 --endpoint=unix://tmp/csi.sock --kubeconfig=/path/to/kubeconfig
```

### StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: btrfs-local
provisioner: btrfs.csi.k8s.io
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
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

## Development

### Project Structure

```
btrfs-csi/
├── main.go                    # Entry point
├── internal/
│   └── driver/
│       ├── driver.go         # CSI driver implementation
│       └── btrfs.go         # Btrfs management functions
├── deploy/
│   └── kubernetes/          # Kubernetes manifests
├── Dockerfile               # Container image
├── Makefile                # Build and deployment scripts
└── README.md               # This file
```

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
make logs
```

### Check Driver Status

```bash
make status
```

### Common Issues

1. **Btrfs not available**: Ensure `btrfs-progs` is installed on all nodes
2. **Permission denied**: The driver needs `SYS_ADMIN` capability for mount operations
3. **Volume not created**: Check if the node has sufficient disk space and Btrfs support

## Security Considerations

- The driver runs with `privileged: true` and `SYS_ADMIN` capability
- This is required for Btrfs subvolume operations and mount operations
- Consider using Pod Security Standards in production environments

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the Apache License 2.0.