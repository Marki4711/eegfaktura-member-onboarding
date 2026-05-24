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
`app.kubernetes.io/version` mappt auf `.Chart.AppVersion` statt auf den
Backend-Image-Tag — vorher landete `sha-XXXXXX` als Version-Label auch
auf Postgres, Frontend und kubectl-CronJob (irreführend, weil die ein
eigenes Image haben). Image-spezifische Tags stehen ohnehin auf den
einzelnen Pods (`image: registry/...:tag`).
*/}}
{{- define "member-onboarding.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/*
Namespace shorthand.
*/}}
{{- define "member-onboarding.namespace" -}}
{{ .Values.namespace }}
{{- end }}
