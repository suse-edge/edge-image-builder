apiVersion: content.hauler.cattle.io/v1alpha1
kind: Images
metadata:
  name: embedded-registry-images
spec:
  images:
    {{- range .ContainerImages }}
    - name: {{ .Name }}
      {{- if .SupplyChainKey }}
      key: {{ .SupplyChainKey }}
      {{- end }}
    {{- end }}
