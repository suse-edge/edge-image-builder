#!/bin/bash
set -euo pipefail

BASEDIR="$(dirname "$0")"
source ${BASEDIR}/basic-setup.sh

RANCHERLOCKNAMESPACE="default"
RANCHERLOCKCMNAME="rancher-lock"

if [ -z "${RANCHER_FINALPASSWORD}" ]; then
  # If there is no final password, then finish the setup right away
  exit 0
fi

trap 'catch $? $LINENO' EXIT

catch() {
  if [ "$1" != "0" ]; then
    echo "Error $1 occurred on $2"
    ${KUBECTL} delete configmap ${RANCHERLOCKCMNAME} -n ${RANCHERLOCKNAMESPACE}
  fi
}

# Get or create the lock to run all those steps just in a single node
# As the first node is created WAY before the others, this should be enough
# TODO: Investigate if leases is better
if [ $(${KUBECTL} get cm -n ${RANCHERLOCKNAMESPACE} ${RANCHERLOCKCMNAME} -o name | wc -l) -lt 1 ]; then
  ${KUBECTL} create configmap ${RANCHERLOCKCMNAME} -n ${RANCHERLOCKNAMESPACE} --from-literal foo=bar
else
  exit 0
fi

# Wait for rancher to be deployed
while ! ${KUBECTL} wait --for condition=ready -n ${RANCHER_CHART_TARGETNAMESPACE} $(${KUBECTL} get pods -n ${RANCHER_CHART_TARGETNAMESPACE} -l app=rancher -o name) --timeout=10s; do sleep 2 ; done
until ${KUBECTL} get ingress -n ${RANCHER_CHART_TARGETNAMESPACE} rancher > /dev/null 2>&1; do sleep 10; done

RANCHERBOOTSTRAPPASSWORD=$(${KUBECTL} get secret -n ${RANCHER_CHART_TARGETNAMESPACE} bootstrap-secret -o jsonpath='{.data.bootstrapPassword}' | base64 -d)
RANCHERHOSTNAME=$(${KUBECTL} get ingress -n ${RANCHER_CHART_TARGETNAMESPACE} rancher -o jsonpath='{.spec.rules[0].host}')

# Skip the whole process if things have been set already
if [ -z $(${KUBECTL} get settings.management.cattle.io first-login -ojsonpath='{.value}') ]; then
  # Add the protocol
  RANCHERHOSTNAME="https://${RANCHERHOSTNAME}"
  TOKEN=""
  while [ -z "${TOKEN}" ]; do
    # Get token
    sleep 2
    TOKEN=$(curl -sk -X POST ${RANCHERHOSTNAME}/v3-public/localProviders/local?action=login -H 'content-type: application/json' -d "{\"username\":\"admin\",\"password\":\"${RANCHERBOOTSTRAPPASSWORD}\"}" | jq -r .token)
  done

  # Set password
  curl -sk ${RANCHERHOSTNAME}/v3/users?action=changepassword -H 'content-type: application/json' -H "Authorization: Bearer $TOKEN" -d "{\"currentPassword\":\"${RANCHERBOOTSTRAPPASSWORD}\",\"newPassword\":\"${RANCHER_FINALPASSWORD}\"}"

  # Create a temporary API token (ttl=60 minutes)
  APITOKEN=$(curl -sk ${RANCHERHOSTNAME}/v3/token -H 'content-type: application/json' -H "Authorization: Bearer ${TOKEN}" -d '{"type":"token","description":"automation","ttl":3600000}' | jq -r .token)

  curl -sk ${RANCHERHOSTNAME}/v3/settings/server-url -H 'content-type: application/json' -H "Authorization: Bearer ${APITOKEN}" -X PUT -d "{\"name\":\"server-url\",\"value\":\"${RANCHERHOSTNAME}\"}"
  curl -sk ${RANCHERHOSTNAME}/v3/settings/telemetry-opt -X PUT -H 'content-type: application/json' -H 'accept: application/json' -H "Authorization: Bearer ${APITOKEN}" -d '{"value":"out"}'
fi

# Clean up the lock cm
${KUBECTL} delete configmap ${RANCHERLOCKCMNAME} -n ${RANCHERLOCKNAMESPACE}
