{{/*
Expand the name of the chart.
*/}}
{{- define "crabby.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "crabby.fullname" -}}
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
{{- define "crabby.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "crabby.labels" -}}
helm.sh/chart: {{ include "crabby.chart" . }}
{{ include "crabby.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "crabby.selectorLabels" -}}
app.kubernetes.io/name: {{ include "crabby.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "crabby.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "crabby.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Whether any storage backends require secrets mounted as files.
*/}}
{{- define "crabby.needsSecrets" -}}
{{- if and .Values.storage.influxdb.enabled (or .Values.storage.influxdb.token .Values.storage.influxdb.existingSecret) -}}
true
{{- else if and .Values.storage.splunkHec.enabled (or .Values.storage.splunkHec.token .Values.storage.splunkHec.existingSecret) -}}
true
{{- else if and .Values.storage.pagerduty.enabled (or .Values.storage.pagerduty.routingKey .Values.storage.pagerduty.existingSecret) -}}
true
{{- end -}}
{{- end }}

{{/*
Name of the secret to use for a given backend.
*/}}
{{- define "crabby.secretName" -}}
{{- .existingSecret | default .fullname }}
{{- end }}

{{/*
Minimum job interval in seconds across all configured jobs.
Falls back to 60 if no jobs are defined.
*/}}
{{- define "crabby.minJobInterval" -}}
{{- $min := 60 }}
{{- range .Values.jobs }}
  {{- if lt (int .interval) $min }}
    {{- $min = int .interval }}
  {{- end }}
{{- end }}
{{- $min }}
{{- end }}
