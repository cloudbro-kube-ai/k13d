{{/*
Expand the name of the chart.
*/}}
{{- define "k13d.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "k13d.fullname" -}}
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
{{- define "k13d.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "k13d.labels" -}}
helm.sh/chart: {{ include "k13d.chart" . }}
{{ include "k13d.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "k13d.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k13d.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "k13d.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "k13d.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the API key secret name
*/}}
{{- define "k13d.apiKeySecretName" -}}
{{- if .Values.k13d.ai.existingSecret }}
{{- .Values.k13d.ai.existingSecret }}
{{- else }}
{{- include "k13d.fullname" . }}-api-key
{{- end }}
{{- end }}

{{/*
Get the database password secret name
*/}}
{{- define "k13d.dbSecretName" -}}
{{- if .Values.k13d.database.existingSecret }}
{{- .Values.k13d.database.existingSecret }}
{{- else }}
{{- include "k13d.fullname" . }}-db
{{- end }}
{{- end }}
