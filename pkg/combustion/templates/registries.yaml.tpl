mirrors:
  docker.io:
    endpoint:
      - "http://localhost:{{ .Port }}"
{{- range .Hostnames }}
  {{ . }}:
    endpoint:
      - "http://localhost:{{ $.Port }}"
{{- end }}