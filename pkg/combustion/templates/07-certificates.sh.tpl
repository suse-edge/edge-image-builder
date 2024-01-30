#!/bin/bash
set -euo pipefail

cp ./{{ .CertificatesDir }}/* /etc/pki/trust/anchors/.
update-ca-certificates -v
