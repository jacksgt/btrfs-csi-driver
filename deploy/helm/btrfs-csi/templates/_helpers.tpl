{{/*
Expand the name of the chart.
*/}}
{{- define "btrfs-csi.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "btrfs-csi.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "btrfs-csi.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "btrfs-csi.labels" -}}
helm.sh/chart: {{ include "btrfs-csi.chart" . }}
{{ include "btrfs-csi.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "btrfs-csi.selectorLabels" -}}
app.kubernetes.io/name: {{ include "btrfs-csi.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "btrfs-csi.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "btrfs-csi.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the CSI driver name
*/}}
{{- define "btrfs-csi.driverName" -}}
{{- printf "btrfs.csi.k8s.io" }}
{{- end }}

{{/*
Create the full image name for the CSI plugin
*/}}
{{- define "btrfs-csi.pluginImage" -}}
{{- printf "%s:%s" .Values.csiPlugin.image.repository .Values.csiPlugin.image.tag }}
{{- end }}

{{/*
Create the full image name for the CSI provisioner
*/}}
{{- define "btrfs-csi.provisionerImage" -}}
{{- printf "%s:%s" .Values.csiProvisioner.image.repository .Values.csiProvisioner.image.tag }}
{{- end }}

{{/*
Create the full image name for the CSI resizer
*/}}
{{- define "btrfs-csi.resizerImage" -}}
{{- printf "%s:%s" .Values.csiResizer.image.repository .Values.csiResizer.image.tag }}
{{- end }}

{{/*
Create the full image name for the CSI node driver registrar
*/}}
{{- define "btrfs-csi.nodeDriverRegistrarImage" -}}
{{- printf "%s:%s" .Values.csiNodeDriverRegistrar.image.repository .Values.csiNodeDriverRegistrar.image.tag }}
{{- end }}
