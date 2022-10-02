#!/usr/bin/env bash
{
set -euo pipefail
set -x

sock="/tmp/syringe.sock"
creds=(
	cred1
	cred2
	cred3
	foobarbaz
)

cred_opts=()
for cred in "${creds[@]}"; do
	cred_opts+=(--property=LoadCredential="${cred}":"${sock}")
done

systemd-run --pty --same-dir --service-type=simple \
	--property=Delegate=yes \
	--property=KillMode=process \
	--property=ExecReload='/usr/local/bin/syringe update /bin/sh -c "env; ls /proc/$$$$/fd"' \
	--property=SetCredential='xyz:test' \
	--property=PrivateMounts=yes \
	--property=User="${USER}" \
	--collect \
	--unit=secrets-test \
	"${cred_opts[@]}" \
	/usr/bin/env bash -exc 'c="${CREDENTIALS_DIRECTORY}/"; while :; do date +%s; ls "$c"; sleep 5; done'
}
