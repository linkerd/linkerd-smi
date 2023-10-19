{{- define "partials.namespace" -}}
{{ if eq .Release.Service "CLI" }}namespace: {{.Release.Namespace}}{{ end }}
{{- end -}}

{{- define "partials.image-pull-secrets"}}
{{- if . }}
imagePullSecrets:
{{ toYaml . | indent 2 }}
{{- end }}
{{- end -}}
