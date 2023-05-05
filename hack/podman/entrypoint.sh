#!/usr/bin/env bash
set -euo pipefail
set -x

trap "poweroff" EXIT

request_cred () {
	local name="${1}"

	#syringe request --socket /run/syringe/syringe.sock --unit "secrets-test.service" --credential "${name}"
	systemd-run -GPdq -u secrets-test --service-type=oneshot --property=LoadCredential="${name}":/run/syringe/syringe.sock \
		/bin/bash -exc 'echo "$(< ${CREDENTIALS_DIRECTORY}/'"${name}"')"'
}

systemctl enable --now syringe.socket

request_cred "foobarbaz"
request_cred "xyz"
request_cred "cred1"
