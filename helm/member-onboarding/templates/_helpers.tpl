{{/*
fullname — used as prefix for all resource names.
Derived from the Helm release name, e.g. "eegfaktura-member-onboarding".
This ensures no name collisions when co-deployed in a shared namespace.
*/}}
{{- define "member-onboarding.fullname" -}}
{{ .Release.Name }}
{{- end }}

{{/*
Common labels applied to every resource.
*/}}
{{- define "member-onboarding.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/version: {{ .Values.images.backend.tag | quote }}
{{- end }}

{{/*
Namespace shorthand.
*/}}
{{- define "member-onboarding.namespace" -}}
{{ .Values.namespace }}
{{- end }}
