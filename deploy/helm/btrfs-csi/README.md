# Btrfs CSI Driver Helm Chart

This Helm chart deploys the Btrfs CSI Driver to a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Btrfs filesystem support on worker nodes

## Installation

### Add the Helm repository (if published)
```bash
helm repo add btrfs-csi https://charts.example.com/btrfs-csi
helm repo update
```

### Install the chart
```bash
helm install btrfs-csi ./deploy/helm/btrfs-csi
```

### Install with custom values
```bash
helm install btrfs-csi ./deploy/helm/btrfs-csi -f custom-values.yaml
```

## Configuration

The following table lists the configurable parameters and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `nameOverride` | Override the name of the chart | `""` |
| `fullnameOverride` | Override the full name of the chart | `""` |
| `serviceAccount.create` | Specifies whether a service account should be created | `true` |
| `serviceAccount.annotations` | Annotations to add to the service account | `{}` |
| `serviceAccount.name` | The name of the service account to use | `""` |
| `csiPlugin.image.repository` | CSI plugin image repository | `ghcr.io/jacksgt/btrfs-csi-driver/plugin` |
| `csiPlugin.image.tag` | CSI plugin image tag | `latest` |
| `csiPlugin.image.pullPolicy` | CSI plugin image pull policy | `IfNotPresent` |
| `csiPlugin.resources` | Resource requests and limits for CSI plugin | `{}` |
| `csiProvisioner.image.repository` | CSI provisioner image repository | `registry.k8s.io/sig-storage/csi-provisioner` |
| `csiProvisioner.image.tag` | CSI provisioner image tag | `v5.3.0` |
| `csiProvisioner.image.pullPolicy` | CSI provisioner image pull policy | `IfNotPresent` |
| `csiProvisioner.resources` | Resource requests and limits for CSI provisioner | `{}` |
| `csiResizer.image.repository` | CSI resizer image repository | `registry.k8s.io/sig-storage/csi-resizer` |
| `csiResizer.image.tag` | CSI resizer image tag | `v1.13.0` |
| `csiResizer.image.pullPolicy` | CSI resizer image pull policy | `IfNotPresent` |
| `csiResizer.resources` | Resource requests and limits for CSI resizer | `{}` |
| `csiNodeDriverRegistrar.image.repository` | CSI node driver registrar image repository | `registry.k8s.io/sig-storage/csi-node-driver-registrar` |
| `csiNodeDriverRegistrar.image.tag` | CSI node driver registrar image tag | `v2.14.0` |
| `csiNodeDriverRegistrar.image.pullPolicy` | CSI node driver registrar image pull policy | `IfNotPresent` |
| `csiNodeDriverRegistrar.resources` | Resource requests and limits for CSI node driver registrar | `{}` |
| `daemonset.nodeSelector` | Node selector for the DaemonSet | `{}` |
| `daemonset.affinity` | Affinity for the DaemonSet | `{}` |
| `daemonset.priorityClassName` | Priority class name for the DaemonSet | `system-cluster-critical` |
| `daemonset.tolerations` | Tolerations for the DaemonSet | `[{effect: NoSchedule, key: node-role.kubernetes.io/master, operator: Exists}]` |
| `daemonset.resources` | Resource requests and limits for the DaemonSet | `{}` |
| `daemonset.updateStrategy` | Update strategy for the DaemonSet | `{}` |
| `daemonset.podAnnotations` | Annotations to add to the DaemonSet pods | `{}` |
| `storageClasses` | List of storage classes to create | See values.yaml |

## Storage Classes

The chart creates storage classes based on the `storageClasses` configuration. Each storage class can have:

- `name`: The name of the storage class
- `provisioner`: The CSI driver name (usually `btrfs.csi.k8s.io`)
- `reclaimPolicy`: The reclaim policy (`Delete` or `Retain`)
- `annotations`: Additional annotations
- `isDefaultClass`: Whether this is the default storage class
- `parameters`: Storage class parameters (e.g., `subvolumeRoot`)

## Examples

### Basic installation
```bash
helm install btrfs-csi ./deploy/helm/btrfs-csi
```

### Install with custom storage class
```yaml
# custom-values.yaml
storageClasses:
  - name: btrfs-fast
    provisioner: btrfs.csi.k8s.io
    reclaimPolicy: Delete
    isDefaultClass: true
    parameters:
      subvolumeRoot: /var/lib/btrfs-csi-fast
```

```bash
helm install btrfs-csi ./deploy/helm/btrfs-csi -f custom-values.yaml
```

### Install with resource limits
```yaml
# custom-values.yaml
csiPlugin:
  resources:
    requests:
      memory: "100Mi"
      cpu: "50m"
    limits:
      memory: "200Mi"
      cpu: "200m"
```

## Uninstalling

To uninstall the chart:

```bash
helm uninstall btrfs-csi
```

## Development

To test the chart locally:

```bash
# Validate the chart
helm lint ./deploy/helm/btrfs-csi

# Test template rendering
helm template btrfs-csi ./deploy/helm/btrfs-csi

# Dry run installation
helm install btrfs-csi ./deploy/helm/btrfs-csi --dry-run --debug
```
