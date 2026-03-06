#!/usr/bin/env bash
# E2E test: boot a FCOS VM, deploy syringe, test credential delivery via LoadCredential.
# Uses QEMU savevm/loadvm to cache a booted VM snapshot for fast restarts.
#
# Env vars:
#   TEST_SSH_PORT      SSH port forward (default: 2223)
#   REBUILD_SNAPSHOT   Set to 1 to force snapshot recreation
#   KEEP_VM            Set to 1 to keep VM running after tests
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
work_dir="${root}/tmp/vm"
ssh_port="${TEST_SSH_PORT:-2223}"
ssh_key="${root}/hack/dev/dev_ed25519"

snapshot_disk="${work_dir}/fcos-snapshot.qcow2"
snapshot_name="ssh-ready"
snapshot_hash_file="${work_dir}/snapshot.hash"
monitor_sock="${work_dir}/qemu-monitor.sock"
pid_file="${work_dir}/qemu.pid"

chmod 600 "${ssh_key}"

fh() {
    fcos-harness --work-dir "${work_dir}" "$@"
}
fh_ssh() {
    fh ssh --ssh-key "${ssh_key}" --ssh-port "${ssh_port}" "$@"
}

sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    else
        shasum -a 256 "$1" | cut -d' ' -f1
    fi
}

# -- Build syringe binaries for Linux --
echo ">>> Building syringe for linux..."
syringe_bin="${work_dir}/syringe"
syringe_update_bin="${work_dir}/syringe-update"
mkdir -p "${work_dir}"
GOOS=linux CGO_ENABLED=0 go build -o "${syringe_bin}" ./cmd/syringe
GOOS=linux CGO_ENABLED=0 go build -o "${syringe_update_bin}" ./cmd/syringe-update

# -- Build Ignition config --
echo ">>> Building Ignition config..."
make -C "${root}/hack/init" config.ign
ign="${root}/hack/init/config.ign"

# -- Ensure FCOS base image --
echo ">>> Ensuring FCOS base image..."
fh image

# -- Snapshot caching --
current_hash="$({ sha256 "${ign}"; sha256 "${syringe_bin}"; sha256 "${syringe_update_bin}"; } | sha256 /dev/stdin)"
use_snapshot=false

if [ "${REBUILD_SNAPSHOT:-}" != "1" ] \
    && [ -f "${snapshot_disk}" ] \
    && [ -f "${snapshot_hash_file}" ] \
    && [ "$(cat "${snapshot_hash_file}")" = "${current_hash}" ] \
    && qemu-img snapshot -l "${snapshot_disk}" 2>/dev/null | grep -q "${snapshot_name}"; then
    use_snapshot=true
    echo ">>> Valid VM snapshot found, skipping boot+goss"
fi

if [ "${use_snapshot}" = "false" ]; then
    echo ">>> Creating new VM snapshot..."
    rm -f "${snapshot_disk}" "${snapshot_hash_file}"

    fh disk --base "${work_dir}/fcos.qcow2" --overlay "${snapshot_disk}"

    fh start \
        --disk "${snapshot_disk}" \
        --ignition "${ign}" \
        --ssh-port "${ssh_port}" \
        --hostname syringe-test \
        --serial-log "${work_dir}/serial-snapshot.log" \
        --qmp "${monitor_sock}" \
        --pid-file "${pid_file}"

    cleanup_snapshot() {
        fh stop --pid-file "${pid_file}" 2>/dev/null || true
        rm -f "${monitor_sock}"
    }
    trap cleanup_snapshot EXIT

    echo ">>> Waiting for SSH..."
    fh_ssh --wait 180 -- true

    echo ">>> Deploying syringe binaries..."
    scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
        -P "${ssh_port}" -i "${ssh_key}" \
        "${syringe_bin}" "${syringe_update_bin}" core@127.0.0.1:/tmp/
    fh_ssh -- "sudo install -m 755 /tmp/syringe /usr/local/bin/syringe && rm /tmp/syringe"
    fh_ssh -- "sudo install -m 4755 /tmp/syringe-update /usr/local/bin/syringe-update && rm /tmp/syringe-update"
    fh_ssh -- "sudo chcon -t initrc_exec_t /usr/local/bin/syringe /usr/local/bin/syringe-update"

    echo ">>> Reloading D-Bus and systemd..."
    fh_ssh -- "sudo systemctl reload dbus"
    fh_ssh -- "sudo systemctl daemon-reload"
    fh_ssh -- "sudo systemctl start syringe.socket"
    fh_ssh -- "sudo systemctl start syringe.service"

    echo ">>> Running goss validation..."
    fh goss "${root}/hack/goss.yaml" --ssh-key "${ssh_key}" --ssh-port "${ssh_port}" --retry-timeout-secs 60

    echo ">>> Saving VM snapshot '${snapshot_name}'..."
    fh qmp --socket "${monitor_sock}" savevm "${snapshot_name}"

    echo ">>> Stopping snapshot VM..."
    fh qmp --socket "${monitor_sock}" quit
    sleep 1
    fh stop --pid-file "${pid_file}" 2>/dev/null || true
    rm -f "${monitor_sock}"
    trap - EXIT

    echo "${current_hash}" > "${snapshot_hash_file}"
    echo ">>> Snapshot saved."
fi

# -- Boot from snapshot and run e2e tests --
echo ">>> Booting VM from snapshot..."
fh start \
    --disk "${snapshot_disk}" \
    --ignition "${ign}" \
    --ssh-port "${ssh_port}" \
    --hostname syringe-test \
    --serial-log "${work_dir}/serial-test.log" \
    --loadvm "${snapshot_name}" \
    --pid-file "${pid_file}"

cleanup() {
    echo ">>> Shutting down test VM..."
    fh stop --pid-file "${pid_file}" 2>/dev/null || true
}
trap cleanup EXIT

echo ">>> Waiting for SSH..."
fh_ssh --wait 30 -- true

echo ">>> Restarting syringe after snapshot restore..."
fh_ssh -- "sudo systemctl restart syringe.service"

# -- Test: credential delivery via LoadCredential --
echo ">>> Running syringe-test.service..."
fh_ssh -- "sudo systemctl start syringe-test.service"

echo ">>> Verifying credentials..."
failures=0

greeting="$(fh_ssh -- "sudo cat /tmp/syringe-e2e-greeting")"
expected_greeting="unit=syringe-test.service, credential=test-greeting"
if [ "${greeting}" = "${expected_greeting}" ]; then
    echo "  PASS: test-greeting credential matches"
else
    echo "  FAIL: test-greeting credential"
    echo "    expected: ${expected_greeting}"
    echo "    got:      ${greeting}"
    failures=$((failures + 1))
fi

hostname_cred="$(fh_ssh -- "sudo cat /tmp/syringe-e2e-hostname")"
expected_hostname="$(fh_ssh -- "hostname")"
if [ "${hostname_cred}" = "${expected_hostname}" ]; then
    echo "  PASS: test-hostname credential matches"
else
    echo "  FAIL: test-hostname credential"
    echo "    expected: ${expected_hostname}"
    echo "    got:      ${hostname_cred}"
    failures=$((failures + 1))
fi

# -- Test: credential reload via ExecReload --
echo ">>> Testing credential reload..."
fh_ssh -- "sudo systemctl start syringe-reload-test.service"
sleep 2

reload_cred="$(fh_ssh -- "sudo cat /tmp/syringe-e2e-reload")"
if [ "${reload_cred}" = "before-reload" ]; then
    echo "  PASS: reload test initial credential matches"
else
    echo "  FAIL: reload test initial credential"
    echo "    expected: before-reload"
    echo "    got:      ${reload_cred}"
    failures=$((failures + 1))
fi

fh_ssh -- "echo after-reload | sudo tee /etc/syringe/reload-data > /dev/null"
fh_ssh -- "sudo systemctl reload syringe-reload-test.service"
sleep 2

reload_cred_after="$(fh_ssh -- "sudo cat /tmp/syringe-e2e-reload")"
if [ "${reload_cred_after}" = "after-reload" ]; then
    echo "  PASS: credential reload picked up new content"
else
    echo "  FAIL: credential reload"
    echo "    expected: after-reload"
    echo "    got:      ${reload_cred_after}"
    fh_ssh -- "sudo journalctl -u syringe-reload-test.service --no-pager -n 20" || true
    failures=$((failures + 1))
fi

# -- Test: D-Bus config reload --
echo ">>> Testing D-Bus config reload..."
fh_ssh -- "sudo systemctl stop syringe-reload-test.service" 2>/dev/null || true
fh_ssh -- "echo before-reload | sudo tee /etc/syringe/reload-data > /dev/null"

# Add a new template via config change, then trigger reload via D-Bus
fh_ssh -- "sudo tee /etc/syringe/config.yml > /dev/null" <<'CONF'
---
templates:
  - unit: "syringe-test.service"
    credential:
      - "test-greeting"
    contents: |
      unit={{ unitname }}, credential={{ credentialname }}
  - unit: "syringe-test.service"
    credential:
      - "test-hostname"
    options:
      sandbox_path: "/etc"
    contents: |
      {{ file "/etc/hostname" }}
  - unit: "syringe-reload-test.service"
    credential:
      - "test-data"
    options:
      sandbox_path: "/etc/syringe"
    contents: |
      {{ file "/etc/syringe/reload-data" }}
  - unit: "syringe-dbus-reload-test.service"
    credential:
      - "test-dbus"
    contents: |
      dbus-reload-ok
CONF

fh_ssh -- "sudo busctl call ee.zentria.syringe1.Syringe /ee/zentria/syringe1 ee.zentria.syringe1.Syringe Reload"

# Start a service that uses the new template added after reload
fh_ssh -- "sudo systemd-run --unit=syringe-dbus-reload-test.service --property=Type=oneshot --property='LoadCredential=test-dbus:/run/syringe/syringe.sock' /bin/sh -c 'cp \"\$CREDENTIALS_DIRECTORY/test-dbus\" /tmp/syringe-e2e-dbus-reload'"

dbus_reload_cred="$(fh_ssh -- "sudo cat /tmp/syringe-e2e-dbus-reload")"
if [ "${dbus_reload_cred}" = "dbus-reload-ok" ]; then
    echo "  PASS: D-Bus config reload picked up new template"
else
    echo "  FAIL: D-Bus config reload"
    echo "    expected: dbus-reload-ok"
    echo "    got:      ${dbus_reload_cred}"
    fh_ssh -- "sudo journalctl -u syringe.service --no-pager -n 20" || true
    failures=$((failures + 1))
fi

if [ "${failures}" -gt 0 ]; then
    echo ">>> ${failures} test(s) FAILED"
    echo ">>> Serial log tail:"
    tail -50 "${work_dir}/serial-test.log" || true
    echo ">>> syringe.service journal:"
    fh_ssh -- "sudo journalctl -u syringe.service --no-pager -n 50" || true
    exit 1
fi

echo ">>> All tests passed."

# -- Keep VM running if requested --
if [ "${KEEP_VM:-}" = "1" ]; then
    echo ">>> VM is still running (ssh -p ${ssh_port} core@127.0.0.1)"
    echo ">>> Press Ctrl-C to stop..."
    trap cleanup INT
    wait "$(cat "${pid_file}")"
fi
