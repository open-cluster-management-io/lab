{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "chart.fullname" -}}
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
{{- define "chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common annotations
*/}}
{{- define "chart.annotations" -}}
meta.helm.sh/release-name: {{ .Release.Name | quote }}
meta.helm.sh/release-namespace: {{ .Release.Namespace | quote }}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "chart.labels" -}}
helm.sh/chart: {{ include "chart.chart" . }}
{{ include "chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "chart.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Generate feature gates string
*/}}
{{- define "featureGates" -}}
{{- $dict := .dict -}}
{{- $first := true -}}
{{- range $key, $value := $dict }}
  {{- if not $first }},{{ end }}
  {{- printf "%s=%t" $key $value }}
  {{- $first = false }}
{{- end }}
{{- end }}

{{/*
Format the image name and tag for the given provider.
For managed kubernetes providers, the image tag is suffixed with the provider name.
These images are bundled with provider-specific auth binaries.
For generic kubernetes providers, the image tag is used as is.
This image has no additional binaries bundled, other than clusteradm.
*/}}
{{- define "controller.image" -}}
{{- $baseImage := printf "%s%s:%s" .Values.imageRegistry .Values.image.repository .Values.image.tag -}}
{{- $provider := "" -}}
{{- if and .Values.global .Values.kubernetesProvider -}}
{{- $provider = .Values.kubernetesProvider | lower -}}
{{- end -}}
{{- if or (eq $provider "eks") (eq $provider "gke") -}}
{{- printf "%s-%s" $baseImage $provider -}}
{{- else -}}
{{- $baseImage -}}
{{- end -}}
{{- end -}}