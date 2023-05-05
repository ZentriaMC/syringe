#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
flags=(
	-v "${root}/syringe:/usr/bin/syringe:ro"
	-v "${root}/config.sample.yml:/etc/syringe/config.yml"
	-v "${root}/init/syringe.service:/etc/systemd/system/syringe.service:ro"
	-v "${root}/init/syringe.socket:/etc/systemd/system/syringe.socket:ro"
	-v "${root}/dbus/ee.zentria.syringe1.Syringe.conf:/usr/share/dbus-1/system.d/ee.zentria.syringe1.Syringe.conf:ro"
	-v "${root}/dbus/ee.zentria.syringe1.Syringe.service:/usr/share/dbus-1/system-services/ee.zentria.syringe1.Syringe.service:ro"
	-v "${root}/hack/podman/entrypoint.sh:/entrypoint.sh:ro"
)

podman run --rm -ti --name ubi "${flags[@]}" registry.access.redhat.com/ubi9:latest \
	/usr/lib/systemd/systemd \
	/entrypoint.sh
