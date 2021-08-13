{{- define "partials.namespace" -}}
{{ if eq .Release.Service "CLI" }}namespace: {{.Release.Namespace}}{{ end }}
{{- end -}}
