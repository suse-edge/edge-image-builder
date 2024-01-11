apiVersion: content.hauler.cattle.io/v1alpha1
kind: Images
metadata:
  name: Embedded-Registry-Images
spec:
  images:
    {{- range .ContainerImages }}
    - name: {{ .Name }}
      {{- if .SupplyChainKey }}
      key: {{ .SupplyChainKey }}
      {{- end }}
    {{- end }}
---
apiVersion: content.hauler.cattle.io/v1alpha1
kind: Charts
metadata:
  name: Embedded-Registry-Charts
spec:
  charts:
    {{- range .HelmCharts }}
    - name: {{ .Name }}
      repoURL: {{ .RepoURL }}
      version: {{ .Version }}
    {{- end }}
