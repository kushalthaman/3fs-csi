{{- define "3fs-csi.name" -}}
3fs-csi
{{- end -}}

{{- define "3fs-csi.serviceAccountName" -}}
{{- if .Values.serviceAccount.name }}
{{ .Values.serviceAccount.name }}
{{- else }}
{{ include "3fs-csi.name" . }}
{{- end -}}
{{- end -}}

