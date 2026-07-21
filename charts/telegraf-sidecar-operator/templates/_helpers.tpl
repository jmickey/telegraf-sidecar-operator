{{/*
Expand the name of the chart.
*/}}
{{- define "_helpers.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "_helpers.fullname" -}}
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
{{- define "_helpers.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "_helpers.labels" -}}
helm.sh/chart: {{ include "_helpers.chart" . }}
{{ include "_helpers.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/part-of: telegraf-sidecar-operator
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "_helpers.selectorLabels" -}}
app.kubernetes.io/name: {{ include "_helpers.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Validate the webhook TLS configuration.
*/}}
{{- define "_helpers.validateWebhookTLS" -}}
{{- $tls := .Values.mutatingWebhook.tls }}
{{- $valid := list "helm" "certManager" "custom" }}
{{- if not (has $tls.method $valid) }}
{{- fail (printf "mutatingWebhook.tls.method must be one of %s, got %q" (join ", " $valid) $tls.method) }}
{{- end }}
{{- if eq $tls.method "custom" }}
{{- if not $tls.custom.existingSecret }}
{{- if or (not $tls.custom.cert) (not $tls.custom.key) }}
{{- fail "mutatingWebhook.tls.method=custom requires either custom.existingSecret, or both custom.cert and custom.key" }}
{{- end }}
{{- end }}
{{- if not $tls.custom.caBundle }}
{{- fail "mutatingWebhook.tls.method=custom requires custom.caBundle" }}
{{- end }}
{{- end }}
{{- end }}
