{{/*
Common labels applied to every resource.
*/}}
{{- define "member-onboarding.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/version: {{ .Values.images.backend.tag | quote }}
{{- end }}

{{/*
Namespace shorthand — always taken from values so overrides work consistently.
*/}}
{{- define "member-onboarding.namespace" -}}
{{ .Values.namespace }}
{{- end }}
