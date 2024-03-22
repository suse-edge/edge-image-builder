#!/bin/bash
# Pre-requisites. Cluster already running
export RKE2KUBECTL="/var/lib/rancher/rke2/bin/kubectl"
export K3SKUBECTL="/opt/bin/kubectl"
export RKE2KUBECONFIG="/etc/rancher/rke2/rke2.yaml"
export K3SKUBECONFIG="/etc/rancher/k3s/k3s.yaml"

export SUSECAFILE="/usr/share/pki/trust/anchors/SUSE_Trust_Root.crt.pem"
export SUSECACM="suse-internal-ca"
export SUSECACMNAMESPACE="kube-system"

########################
# METAL3 CHART DETAILS #
########################
export METAL3_CHART_NAME="metal3"
export METAL3_CHART_VERSION="0.6.3"
export METAL3_CHART_VALUESFILE="metal3.yaml"
export METAL3_CHART_CREATENAMESPACE="True"
export METAL3_CHART_INSTALLATIONNAMESPACE="kube-system"
export METAL3_CHART_TARGETNAMESPACE="metal3-system"

###########################
# METAL3 CHART REPOSITORY #
###########################
export METAL3_CHART_REPOSITORY_NAME="suse-edge"
export METAL3_CHART_REPOSITORY_URL="https://suse-edge.github.io/charts"
export METAL3_CHART_REPOSITORY_CAFILE=""
export METAL3_CHART_REPOSITORY_PLAINHTTP="False"
export METAL3_CHART_REPOSITORY_SKIPTLSVERIFY="False"
export METAL3_CHART_REPOSITORY_USERNAME=""
export METAL3_CHART_REPOSITORY_PASSWORD=""

###############
# METAL3 CAPI #
###############
export METAL3_CLUSTERCTLVERSION="1.6.2"
export METAL3_CAPICOREVERSION="1.6.0"
export METAL3_CAPIMETAL3VERSION="1.6.0"
export METAL3_CAPIRKE2VERSION="0.2.6"
export METAL3_CAPIPROVIDER="rke2"
export METAL3_CAPISYSTEMNAMESPACE="capi-system"
export METAL3_RKE2BOOTSTRAPNAMESPACE="rke2-bootstrap-system"
export METAL3_CAPM3NAMESPACE="capm3-system"
export METAL3_RKE2CONTROLPLANENAMESPACE="rke2-control-plane-system"

###########
# METALLB #
###########
export METALLBNAMESPACE="metallb-system"

###########
# RANCHER #
###########
export RANCHER_CHART_NAME="rancher"
export RANCHER_CHART_VERSION="2.8.2"
export RANCHER_CHART_VALUESFILE="rancher.yaml"
export RANCHER_CHART_CREATENAMESPACE="True"
export RANCHER_CHART_INSTALLATIONNAMESPACE="kube-system"
export RANCHER_CHART_TARGETNAMESPACE="cattle-system"

export RANCHER_FINALPASSWORD="adminadminadmin"

############################
# RANCHER CHART REPOSITORY #
############################
export RANCHER_CHART_REPOSITORY_NAME="rancher-stable"
export RANCHER_CHART_REPOSITORY_URL="https://releases.rancher.com/server-charts/stable"
export RANCHER_CHART_REPOSITORY_CAFILE=""
export RANCHER_CHART_REPOSITORY_PLAINHTTP="False"
export RANCHER_CHART_REPOSITORY_SKIPTLSVERIFY="False"
export RANCHER_CHART_REPOSITORY_USERNAME=""
export RANCHER_CHART_REPOSITORY_PASSWORD=""

die(){
  echo ${1} 1>&2
  exit ${2}
}

setup_kubetools(){
  RETRIES=10
  SLEEPTIME=2

  # Identify if K3s or RKE2 (timeout = reties * sleep time)
  t=${RETRIES}
  until [ -e ${RKE2KUBECONFIG} ] || [ -e ${K3SKUBECONFIG} ] && (( t-- > 0 )); do
    sleep ${SLEEPTIME}
  done
  if [ -e "${RKE2KUBECONFIG}" ]; then
    export KUBECONFIG=${RKE2KUBECONFIG}
    export KUBECTL=${RKE2KUBECTL}
  else
    export KUBECONFIG=${K3SKUBECONFIG}
    export KUBECTL=${K3SKUBECTL}
  fi

  # Wait for the node to be available, meaning the K8s API is available
  while ! ${KUBECTL} wait --for condition=ready node $(cat /etc/hostname | tr '[:upper:]' '[:lower:]') ; do sleep 2 ; done

  # https://github.com/rancher/rke2/issues/3958
  if [ "${KUBECTL}" == "${RKE2KUBECTL}" ]; then
    # Wait for the rke2-ingress-nginx-controller DS to be available if using RKE2
    while ! ${KUBECTL} rollout status daemonset -n kube-system rke2-ingress-nginx-controller --timeout=60s; do sleep 2 ; done
  fi
}

setup_suse_internal_ca(){
  # Check if the CA configmap is already available
  if [ $(${KUBECTL} get configmap -n ${SUSECACMNAMESPACE} ${SUSECACM} -o name | wc -l) -lt 1 ]; then
    if [ -f ${SUSECAFILE} ]; then
      # Create the CA
      ${KUBECTL} create cm ${SUSECACM} -n ${SUSECACMNAMESPACE} --from-file=ca.crt=${SUSECAFILE}
    fi
  fi
}

setup_kubetools
setup_suse_internal_ca