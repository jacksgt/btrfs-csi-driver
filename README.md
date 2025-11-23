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

Deploy latest version of btrfs-csi-driver and ensure the pods are running

```sh
kubectl apply -f deploy/kubernetes

kubectl get pod -l app=btrfs-csi-driver
NAME                    READY   STATUS    RESTARTS    AGE
btrfs-csi-vcffm         4/4     Running   0           60s

kubectl get storageclass
NAME             PROVISIONER          RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
btrfs-local      btrfs.csi.k8s.io     Delete          WaitForFirstConsumer   true                   66s
```

The default manifests create a `StorageClass` that points to `/var/lib/btrfs-csi`.
The path for the `StorageClass` must be created manually:

```sh
ssh <node> mkdir /var/lib/brtfs-csi
```

If you want to use capacity management (i.e. set a maximum size for each volume and get metrics about how much each volume uses), ensure that [Btrfs quota groups](https://btrfs.readthedocs.io/en/latest/Qgroups.html) are enabled:

```sh
ssh <node> btrfs quote enable /var/lib/btrfs-csi
```

You can check if qgroups are enabled with the following command:

```sh
ssh <node> btrfs qgroup show /var/lib/btrfs-csi
```

If you want to use a different filesystem or disk that is mounted in a different location, make sure to adjust the paths above as necessary.

Now you can create a PersistentVolumeClaim that makes use of the new StorageClass.
Note that a volume will only be provisioned once a Pod starts using the PVC (`volumeBindingMode: WaitForFirstConsumer`).

## Helm Chart

The Helm chart is recommended for deployment of the Btrfs CSI Driver on Kubernetes.
The chart supports extensive configuration through values.
Please refer to the [Helm chart README](./deploy/helm/btrfs-csi/README.md) for detailed configuration options and examples.

```bash
# Install the chart with default config (see "Quick Start" above)
helm install btrfs-csi oci://ghcr.io/jacksgt/btrfs-csi-driver/charts/btrfs-csi:0.0.4

# Install with custom values and in custom namespace
helm -n kube-system install btrfs-csi -f custom-values.yaml oci://ghcr.io/jacksgt/btrfs-csi-driver/charts/btrfs-csi:0.0.4
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

## Volume Expansion

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

The driver runs with `privileged: true` as well as `SYS_ADMIN` and `SYS_CHROOT` capability, this is required for Btrfs subvolume operations and mount operations.
Furthermore, to allow creating storageclasses in arbitrary locations on the node (`/var/lib/btrfs-csi`, `/mnt/data`, ...), the entire host filesystem is mounted into the plugin container.
This also enables the container to use the host's `btrfs` CLI tool (via chroot).
Consider using Pod Security Standards in production environments.

## License

This project is licensed under the Apache License 2.0.
