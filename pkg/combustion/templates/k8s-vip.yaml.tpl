---
apiVersion: v1
kind: Namespace
metadata:
  name: metallb-system
spec: {}
---
apiVersion: v1
kind: Namespace
metadata:
  name: endpoint-copier-operator
spec: {}
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: metallb
  namespace: metallb-system
spec:
  repo: https://suse-edge.github.io/charts
  chart: metallb
  targetNamespace: metallb-system
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: endpoint-copier-operator
  namespace: endpoint-copier-operator
spec:
  repo: https://suse-edge.github.io/endpoint-copier-operator
  chart: endpoint-copier-operator
  targetNamespace: endpoint-copier-operator
---
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: api-ip
  namespace: metallb-system
spec:
  addresses:
  - {{ .APIAddress }}/32
  avoidBuggyIPs: true
  serviceAllocation:
    namespaces:
      - default
    serviceSelectors:
      - matchExpressions:
        - {key: "serviceType", operator: In, values: [kubernetes-vip]}
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: api-ip-l2-adv
  namespace: metallb-system
spec:
  ipAddressPools:
  - api-ip
---
apiVersion: v1
kind: Service
metadata:
  name: kubernetes-vip
  namespace: default
  labels:
    serviceType: kubernetes-vip
spec:
  ports:
{{- if .RKE2 }}
  - name: rke2-api
    port: 9345
    protocol: TCP
    targetPort: 9345
{{- end }}
  - name: k8s-api
    port: 6443
    protocol: TCP
    targetPort: 6443
  type: LoadBalancer
