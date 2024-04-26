{{/*
Expand the name of the chart.
*/}}
{{- define "pganalyze-collector.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "pganalyze-collector.fullname" -}}
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
{{- define "pganalyze-collector.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "pganalyze-collector.labels" -}}
helm.sh/chart: {{ include "pganalyze-collector.chart" . }}
{{ include "pganalyze-collector.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "pganalyze-collector.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pganalyze-collector.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the config map to use
*/}}
{{- define "pganalyze-collector.configMapName" -}}
{{- if .Values.configMap.create }}
{{- default (include "pganalyze-collector.fullname" .) .Values.configMap.name }}
{{- else }}
{{- .Values.configMap.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the secret to use
*/}}
{{- define "pganalyze-collector.secretName" -}}
{{- if .Values.secret.create }}
{{- default (include "pganalyze-collector.fullname" .) .Values.secret.name }}
{{- else }}
{{- .Values.secret.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "pganalyze-collector.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "pganalyze-collector.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the service to use
*/}}
{{- define "pganalyze-collector.serviceName" -}}
{{- if .Values.service.create }}
{{- default (include "pganalyze-collector.fullname" .) .Values.service.name }}
{{- else }}
{{- .Values.service.name }}
{{- end }}
{{- end }}
