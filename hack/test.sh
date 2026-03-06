#!/usr/bin/env bash
# E2E test: boot a FCOS VM, deploy syringe, test credential delivery via LoadCredential.
# Uses QEMU savevm/loadvm to cache a booted VM snapshot for fast restarts.
#
# Env vars:
#   FCOS_HARNESS_SSH_PORT   SSH port forward (default: 2223)
#   FCOS_HARNESS_SSH_KEY    SSH key (set below)
#   REBUILD_SNAPSHOT        Set to 1 to force snapshot recreation
#   KEEP_VM                 Set to 1 to keep VM running after tests
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
export FCOS_HARNESS_WORK_DIR="${root}/tmp/vm"
export FCOS_HARNESS_SSH_KEY="${root}/hack/dev/dev_ed25519"
export FCOS_HARNESS_SSH_PORT="${FCOS_HARNESS_SSH_PORT:-2223}"

chmod 600 "${FCOS_HARNESS_SSH_KEY}"

fh() { fcos-harness "$@"; }

snapshot_disk="${FCOS_HARNESS_WORK_DIR}/fcos-snapshot.qcow2"
snapshot_name="ssh-ready"
snapshot_hash_file="${FCOS_HARNESS_WORK_DIR}/snapshot.hash"
monitor_sock="${FCOS_HARNESS_WORK_DIR}/qemu-monitor.sock"
pid_file="${FCOS_HARNESS_WORK_DIR}/qemu.pid"

sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    else
        shasum -a 256 "$1" | cut -d' ' -f1
    fi
}

# -- Build syringe binaries for Linux --
echo ">>> Building syringe for linux..."
syringe_bin="${FCOS_HARNESS_WORK_DIR}/syringe"
syringe_update_bin="${FCOS_HARNESS_WORK_DIR}/syringe-update"
mkdir -p "${FCOS_HARNESS_WORK_DIR}"
GOOS=linux CGO_ENABLED=0 go build -o "${syringe_bin}" ./cmd/syringe
GOOS=linux CGO_ENABLED=0 go build -o "${syringe_update_bin}" ./cmd/syringe-update

echo ">>> Fetching age for linux..."
age_version="1.3.1"
age_arch="$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
age_tarball="${FCOS_HARNESS_WORK_DIR}/age-v${age_version}-linux-${age_arch}.tar.gz"
age_bin="${FCOS_HARNESS_WORK_DIR}/age"
if [ ! -f "${age_bin}" ]; then
    curl -sL "https://github.com/FiloSottile/age/releases/download/v${age_version}/age-v${age_version}-linux-${age_arch}.tar.gz" -o "${age_tarball}"
    tar -xzf "${age_tarball}" -C "${FCOS_HARNESS_WORK_DIR}" --strip-components=1 age/age age/age-keygen
    rm -f "${age_tarball}"
fi

# -- Build Ignition config --
echo ">>> Building Ignition config..."
make -C "${root}/hack/init" config.ign
ign="${root}/hack/init/config.ign"

# -- Ensure FCOS base image --
echo ">>> Ensuring FCOS base image..."
fh image

# -- Snapshot caching --
current_hash="$({ sha256 "${ign}"; sha256 "${syringe_bin}"; sha256 "${syringe_update_bin}"; sha256 "${age_bin}"; } | sha256 /dev/stdin)"
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

    fh disk --base "${FCOS_HARNESS_WORK_DIR}/fcos.qcow2" --overlay "${snapshot_disk}"

    fh start \
        --disk "${snapshot_disk}" \
        --ignition "${ign}" \
        --hostname syringe-test \
        --serial-log "${FCOS_HARNESS_WORK_DIR}/serial-snapshot.log" \
        --qmp "${monitor_sock}" \
        --pid-file "${pid_file}"

    trap 'fh down 2>/dev/null || true' EXIT

    echo ">>> Waiting for SSH..."
    fh ssh --wait 180 -- true

    echo ">>> Deploying syringe and age binaries..."
    scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
        -P "${FCOS_HARNESS_SSH_PORT}" -i "${FCOS_HARNESS_SSH_KEY}" \
        "${syringe_bin}" "${syringe_update_bin}" "${age_bin}" "${FCOS_HARNESS_WORK_DIR}/age-keygen" core@127.0.0.1:/tmp/
    fh ssh -- "sudo install -m 755 /tmp/syringe /usr/local/bin/syringe && rm /tmp/syringe"
    fh ssh -- "sudo install -m 4755 /tmp/syringe-update /usr/local/bin/syringe-update && rm /tmp/syringe-update"
    fh ssh -- "sudo install -m 755 /tmp/age /usr/local/bin/age && rm /tmp/age"
    fh ssh -- "sudo install -m 755 /tmp/age-keygen /usr/local/bin/age-keygen && rm /tmp/age-keygen"
    fh ssh -- "sudo chcon -t initrc_exec_t /usr/local/bin/syringe /usr/local/bin/syringe-update"

    echo ">>> Reloading D-Bus and systemd..."
    fh ssh -- "sudo systemctl reload dbus"
    fh ssh -- "sudo systemctl daemon-reload"
    fh ssh -- "sudo systemctl start syringe.socket"
    fh ssh -- "sudo systemctl start syringe.service"

    echo ">>> Running goss validation..."
    fh goss "${root}/hack/goss.yaml" --retry-timeout-secs 60

    echo ">>> Saving VM snapshot '${snapshot_name}'..."
    fh qmp --socket "${monitor_sock}" savevm "${snapshot_name}"

    echo ">>> Stopping snapshot VM..."
    fh qmp --socket "${monitor_sock}" quit
    sleep 1
    fh down 2>/dev/null || true
    trap - EXIT

    echo "${current_hash}" > "${snapshot_hash_file}"
    echo ">>> Snapshot saved."
fi

# -- Boot from snapshot and run e2e tests --
echo ">>> Booting VM from snapshot..."
fh start \
    --disk "${snapshot_disk}" \
    --ignition "${ign}" \
    --hostname syringe-test \
    --serial-log "${FCOS_HARNESS_WORK_DIR}/serial-test.log" \
    --loadvm "${snapshot_name}" \
    --pid-file "${pid_file}"

trap 'fh down' EXIT

echo ">>> Waiting for SSH..."
fh ssh --wait 30 -- true

echo ">>> Restarting syringe after snapshot restore..."
fh ssh -- "sudo systemctl restart syringe.service"

# -- Test: credential delivery via LoadCredential --
echo ">>> Running syringe-test.service..."
fh ssh -- "sudo systemctl start syringe-test.service"

echo ">>> Verifying credentials..."
failures=0

greeting="$(fh ssh -- "sudo cat /tmp/syringe-e2e-greeting")"
expected_greeting="unit=syringe-test.service, credential=test-greeting"
if [ "${greeting}" = "${expected_greeting}" ]; then
    echo "  PASS: test-greeting credential matches"
else
    echo "  FAIL: test-greeting credential"
    echo "    expected: ${expected_greeting}"
    echo "    got:      ${greeting}"
    failures=$((failures + 1))
fi

hostname_cred="$(fh ssh -- "sudo cat /tmp/syringe-e2e-hostname")"
expected_hostname="$(fh ssh -- "hostname")"
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
fh ssh -- "sudo systemctl start syringe-reload-test.service"
sleep 2

reload_cred="$(fh ssh -- "sudo cat /tmp/syringe-e2e-reload")"
if [ "${reload_cred}" = "before-reload" ]; then
    echo "  PASS: reload test initial credential matches"
else
    echo "  FAIL: reload test initial credential"
    echo "    expected: before-reload"
    echo "    got:      ${reload_cred}"
    failures=$((failures + 1))
fi

fh ssh -- "echo after-reload | sudo tee /etc/syringe/reload-data > /dev/null"
fh ssh -- "sudo systemctl reload syringe-reload-test.service"
sleep 2

reload_cred_after="$(fh ssh -- "sudo cat /tmp/syringe-e2e-reload")"
if [ "${reload_cred_after}" = "after-reload" ]; then
    echo "  PASS: credential reload picked up new content"
else
    echo "  FAIL: credential reload"
    echo "    expected: after-reload"
    echo "    got:      ${reload_cred_after}"
    fh ssh -- "sudo journalctl -u syringe-reload-test.service --no-pager -n 20" || true
    failures=$((failures + 1))
fi

# -- Test: D-Bus config reload --
echo ">>> Testing D-Bus config reload..."
fh ssh -- "sudo systemctl stop syringe-reload-test.service" 2>/dev/null || true
fh ssh -- "echo before-reload | sudo tee /etc/syringe/reload-data > /dev/null"

# Add a new template via config change, then trigger reload via D-Bus
fh ssh -- "sudo tee /etc/syringe/config.yml > /dev/null" <<'CONF'
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

fh ssh -- "sudo busctl call ee.zentria.syringe1.Syringe /ee/zentria/syringe1 ee.zentria.syringe1.Syringe Reload"

# Start a service that uses the new template added after reload
fh ssh -- "sudo systemd-run --unit=syringe-dbus-reload-test.service --property=Type=oneshot --property='LoadCredential=test-dbus:/run/syringe/syringe.sock' /bin/sh -c 'cp \"\$CREDENTIALS_DIRECTORY/test-dbus\" /tmp/syringe-e2e-dbus-reload'"

dbus_reload_cred="$(fh ssh -- "sudo cat /tmp/syringe-e2e-dbus-reload")"
if [ "${dbus_reload_cred}" = "dbus-reload-ok" ]; then
    echo "  PASS: D-Bus config reload picked up new template"
else
    echo "  FAIL: D-Bus config reload"
    echo "    expected: dbus-reload-ok"
    echo "    got:      ${dbus_reload_cred}"
    fh ssh -- "sudo journalctl -u syringe.service --no-pager -n 20" || true
    failures=$((failures + 1))
fi

# -- Test: age decryption via SSH host key --
echo ">>> Testing age decryption..."

# Get the VM's SSH host public key and encrypt a test secret on the host
ssh_host_pubkey="${FCOS_HARNESS_WORK_DIR}/vm_host_ed25519.pub"
fh ssh -- "cat /etc/ssh/ssh_host_ed25519_key.pub" > "${ssh_host_pubkey}"

age_secret="age-decryption-e2e-ok"
age_encrypted="${FCOS_HARNESS_WORK_DIR}/age-test-secret.age"
echo -n "${age_secret}" | age -R "${ssh_host_pubkey}" -o "${age_encrypted}"

# Deploy the encrypted file to the VM
scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
    -P "${FCOS_HARNESS_SSH_PORT}" -i "${FCOS_HARNESS_SSH_KEY}" \
    "${age_encrypted}" core@127.0.0.1:/tmp/age-test-secret.age
fh ssh -- "sudo install -m 644 /tmp/age-test-secret.age /etc/syringe/age-test-secret.age && rm /tmp/age-test-secret.age"

# Reload syringe with age identity + template
fh ssh -- "sudo tee /etc/syringe/config.yml > /dev/null" <<'CONF'
---
age:
  identities:
    - /etc/ssh/ssh_host_ed25519_key

templates:
  - unit: "syringe-age-test.service"
    credential:
      - "test-age"
    options:
      sandbox_path: "/etc/syringe"
    contents: |
      {{ age "/etc/syringe/age-test-secret.age" }}
CONF

fh ssh -- "sudo busctl call ee.zentria.syringe1.Syringe /ee/zentria/syringe1 ee.zentria.syringe1.Syringe Reload"

fh ssh -- "sudo systemd-run --unit=syringe-age-test.service --property=Type=oneshot --property='LoadCredential=test-age:/run/syringe/syringe.sock' /bin/sh -c 'cp \"\$CREDENTIALS_DIRECTORY/test-age\" /tmp/syringe-e2e-age'"

age_cred="$(fh ssh -- "sudo cat /tmp/syringe-e2e-age")"
if [ "${age_cred}" = "${age_secret}" ]; then
    echo "  PASS: age decryption with SSH host key"
else
    echo "  FAIL: age decryption"
    echo "    expected: ${age_secret}"
    echo "    got:      ${age_cred}"
    fh ssh -- "sudo journalctl -u syringe.service --no-pager -n 20" || true
    failures=$((failures + 1))
fi

# -- Test: age decryption with JSON --
echo ">>> Testing age JSON decryption..."

age_json_encrypted="${FCOS_HARNESS_WORK_DIR}/age-test-json.age"
echo -n '{"db_host":"localhost","db_pass":"s3cret"}' | age -R "${ssh_host_pubkey}" -o "${age_json_encrypted}"

scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR \
    -P "${FCOS_HARNESS_SSH_PORT}" -i "${FCOS_HARNESS_SSH_KEY}" \
    "${age_json_encrypted}" core@127.0.0.1:/tmp/age-test-json.age
fh ssh -- "sudo install -m 644 /tmp/age-test-json.age /etc/syringe/age-test-json.age && rm /tmp/age-test-json.age"

fh ssh -- "sudo tee /etc/syringe/config.yml > /dev/null" <<'CONF'
---
age:
  identities:
    - /etc/ssh/ssh_host_ed25519_key

templates:
  - unit: "syringe-age-json-test.service"
    credential:
      - "test-age-json"
    options:
      sandbox_path: "/etc/syringe"
    contents: |
      {{ (age "/etc/syringe/age-test-json.age" | sprig_fromJson).db_pass }}
CONF

fh ssh -- "sudo busctl call ee.zentria.syringe1.Syringe /ee/zentria/syringe1 ee.zentria.syringe1.Syringe Reload"

fh ssh -- "sudo systemd-run --unit=syringe-age-json-test.service --property=Type=oneshot --property='LoadCredential=test-age-json:/run/syringe/syringe.sock' /bin/sh -c 'cp \"\$CREDENTIALS_DIRECTORY/test-age-json\" /tmp/syringe-e2e-age-json'"

age_json_cred="$(fh ssh -- "sudo cat /tmp/syringe-e2e-age-json")"
if [ "${age_json_cred}" = "s3cret" ]; then
    echo "  PASS: age JSON decryption with SSH host key"
else
    echo "  FAIL: age JSON decryption"
    echo "    expected: s3cret"
    echo "    got:      ${age_json_cred}"
    fh ssh -- "sudo journalctl -u syringe.service --no-pager -n 20" || true
    failures=$((failures + 1))
fi

if [ "${failures}" -gt 0 ]; then
    echo ">>> ${failures} test(s) FAILED"
    echo ">>> Serial log tail:"
    tail -50 "${FCOS_HARNESS_WORK_DIR}/serial-test.log" || true
    echo ">>> syringe.service journal:"
    fh ssh -- "sudo journalctl -u syringe.service --no-pager -n 50" || true
    exit 1
fi

echo ">>> All tests passed."

# -- Keep VM running if requested --
if [ "${KEEP_VM:-}" = "1" ]; then
    echo ">>> VM is still running (ssh -p ${FCOS_HARNESS_SSH_PORT} core@127.0.0.1)"
    echo ">>> Press Ctrl-C to stop..."
    trap 'fh down; exit 0' INT
    while kill -0 "$(cat "${pid_file}")" 2>/dev/null; do
        sleep 1
    done
fi
