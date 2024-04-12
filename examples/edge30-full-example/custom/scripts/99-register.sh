#!/bin/bash
set -euo pipefail

# Registration https://www.suse.com/support/kb/doc/?id=000018564
if ! which SUSEConnect > /dev/null 2>&1; then
	zypper --non-interactive install suseconnect-ng
fi
SUSEConnect --email "<EMAIL>" --url "https://scc.suse.com" --regcode "<REGCODE>"
