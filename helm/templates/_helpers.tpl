{{/*
Expand the name of the chart.
*/}}
{{- define "cluster-network-policy-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "cluster-network-policy-operator.fullname" -}}
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
{{- define "cluster-network-policy-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "cluster-network-policy-operator.labels" -}}
{{- if ne .Chart.Version "0.0.0" -}}
helm.sh/chart: {{ include "cluster-network-policy-operator.chart" . }}
{{ end -}}
{{ include "cluster-network-policy-operator.selectorLabels" . }}
app.kubernetes.io/version: {{ (default .Values.image.tag .Chart.AppVersion) | quote }}
{{- if ne .Chart.Version "0.0.0" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "cluster-network-policy-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "cluster-network-policy-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "cluster-network-policy-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (printf "%s-manager" (include "cluster-network-policy-operator.fullname" .)) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
ClusterRole name
*/}}
{{- define "cluster-network-policy-operator.clusterRoleName" -}}
{{- include "cluster-network-policy-operator.fullname" . }}-manager-role
{{- end }}

{{/*
ClusterRole metadata
*/}}
{{- define "cluster-network-policy-operator.clusterRoleMetadata" -}}
name: {{ include "cluster-network-policy-operator.clusterRoleName" . }}
labels:
{{- include "cluster-network-policy-operator.labels" . | nindent 2 }}
{{- end }}

{{/*
Role name
*/}}
{{- define "cluster-network-policy-operator.roleName" -}}
{{- include "cluster-network-policy-operator.fullname" . }}-leader-election-role
{{- end }}

{{/*
Service name
*/}}
{{- define "cluster-network-policy-operator.serviceName" -}}
{{- default (printf "%s-metrics" (include "cluster-network-policy-operator.fullname" .)) .Values.metrics.service.name }}
{{- end }}

{{/*
Manager image
*/}}
{{- define "cluster-network-policy-operator.managerImage" -}}
{{ printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) }}
{{- end }}

{{/*
Manager arguments
*/}}
{{- define "cluster-network-policy-operator.managerArgs" -}}
- "--leader-elect"
- "--health-probe-bind-address=:8081"
{{- if .Values.metrics.enable }}
- "--metrics-bind-address=:8080"
{{- else }}
- "--metrics-bind-address=0"
{{- end }}
- {{ printf "--exclude-namespaces=%s" (include "cluster-network-policy-operator.join-namespaces" (dict "list" .Values.operator.namespaces.exclude "default" .Release.Namespace)) | quote }}
- {{ printf "--include-namespaces=%s" (include "cluster-network-policy-operator.join-namespaces" (dict "list" .Values.operator.namespaces.include "default" .Release.Namespace)) | quote }}
{{- range .Values.operator.additionalArguments }}
- {{ . | quote }}
{{- end }}
{{- end }}

{{/*
Manager ports
*/}}
{{- define "cluster-network-policy-operator.managerPorts" -}}
{{- if .Values.metrics.enable -}}
- containerPort: 8080
  protocol: TCP
{{ end -}}
- containerPort: 8081
  protocol: TCP
{{- end }}

{{/*
Join namespaces
*/}}
{{- define "cluster-network-policy-operator.join-namespaces" -}}
{{- $namespaces := list -}}
{{- range .list -}}
{{- $namespaces = append $namespaces (default $.default .) -}}
{{- end -}}
{{- $namespaces | join "," -}}
{{- end }}
