#!/usr/bin/env bash
set -euo pipefail
set -x

sudo rm /tmp/syringe.sock || true

export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_NAMESPACE=""
export VAULT_TOKEN="foobarbaz"

sd=(
    systemd-run --pty --same-dir
    --service-type=dbus
    --property=BusName=ee.zentria.syringe1.Syringe
    --property=KillMode=process
    --collect
    --unit=syringe
)
nix-shell -p go --command 'go build -o /tmp/syringe ./cmd/syringe && sudo mv /tmp/syringe /usr/local/bin/syringe && sudo chown root:root /usr/local/bin/syringe && sudo chmod 6755 /usr/local/bin/syringe && exec sudo -E '"${sd[*]}"' /usr/local/bin/syringe /tmp/syringe.sock'
