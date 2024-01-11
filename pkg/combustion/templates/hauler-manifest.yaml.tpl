apiVersion: content.hauler.cattle.io/v1alpha1
kind: Images
metadata:
  name: hauler-content-images-example
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
  name: hauler-content-charts-example
spec:
  charts:
    {{- range .HelmCharts }}
    - name: {{ .Name }}
      repoURL: {{ .RepoURL }}
      version: {{ .Version }}
    {{- end }}
