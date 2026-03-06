# Syringe

Credential provider for systemd's [`LoadCredential=`][systemd-load-credential] Unix socket protocol.

Syringe renders Go templates and serves the output over a Unix socket. When a systemd unit specifies `LoadCredential=name:/run/syringe/syringe.sock`, systemd connects to the socket and receives the rendered credential.

## Features

- **Template-based credentials** — Go templates with built-in functions for files, Vault secrets, [Sprig][sprig], [sockaddr][sockaddr], hashing, base64, and more
- **HashiCorp Vault integration** — read secrets from Vault and render them into credentials
- **File-based credentials** — read from local files with path sandboxing
- **Configuration reload** — SIGHUP reloads the config without restarting; also exposed as a D-Bus method (`ee.zentria.syringe1.Syringe.Reload`)
- **Credential update** — `syringe update` re-fetches credentials for a running service via mount namespace jumping (used as `ExecReload=`), working around [systemd/systemd#21099][systemd-issue-21099]
- **D-Bus service** — exposes `GetSocketPaths` and `Reload` methods on the system bus
- **systemd socket activation** — can be started on-demand via `syringe.socket`

## Configuration

Syringe uses a YAML configuration file (default: `/etc/syringe/config.yml`):

```yaml
templates:
  - unit: "myapp.service"
    credential:
      - "db-password"
    contents: |
      {{ (vault_read "secret/data/myapp").Data.data.db_password }}

  - unit: "myapp.service"
    credential:
      - "hostname"
    options:
      sandbox_path: "/etc"
    contents: |
      {{ file "/etc/hostname" }}

  - unit: "myapp.service"
    # omit credential list for catch-all (matches any credential name)
    contents: |
      unit={{ unitname }}, credential={{ credentialname }}
```

### Template functions

| Function | Description |
|---|---|
| `unitname` | Requesting unit name |
| `credentialname` | Requested credential name |
| `file "<path>"` | Read a file (subject to `sandbox_path`) |
| `vault_read "<path>"` | Read a Vault secret |
| `b64encode` / `b64decode` | Base64 encoding/decoding |
| `sha256sum` / `sha1sum` / `md5sum` | Hash functions |
| `time [format] [modifier]` | Current time (unix, rfc3339, or Go format) |
| `sockaddr "<expr>"` | [go-sockaddr][sockaddr] template |
| `sprig_*` | All [Sprig][sprig] functions (prefixed with `sprig_`) |

### Template options

| Option | Description |
|---|---|
| `delim_left` / `delim_right` | Custom template delimiters (default: `{{` / `}}`) |
| `sandbox_path` | Restrict `file` reads to this directory |
| `allow_missing` | Treat missing template keys as empty instead of erroring |

## Usage

```bash
# Run the server
syringe server --config /etc/syringe/config.yml

# Request a credential manually
syringe request --socket /run/syringe/syringe.sock --unit myapp.service --credential db-password

# Update credentials for the calling service (used as ExecReload=)
syringe update
```

## Deployment

Syringe runs as a systemd service with socket activation:

```ini
# /usr/lib/systemd/system/syringe.socket
[Socket]
ListenStream=/run/syringe/syringe.sock
SocketMode=0600
SocketUser=root
SocketGroup=root

[Install]
WantedBy=sockets.target
```

```ini
# /usr/lib/systemd/system/syringe.service
[Service]
Type=dbus
BusName=ee.zentria.syringe1.Syringe
ExecStart=/usr/bin/syringe server
```

```ini
# /usr/share/dbus-1/system.d/ee.zentria.syringe1.Syringe.conf
# D-Bus policy — see dbus/ directory for the full file
```

```ini
# /usr/share/dbus-1/system-services/ee.zentria.syringe1.Syringe.service
# D-Bus activation service — see dbus/ directory for the full file
```

On Red Hat derived SELinux-enabled systems, the binary needs the `initrc_exec_t` label for `LoadCredential` socket access:

```bash
chcon -t initrc_exec_t /usr/bin/syringe
```

## Credential reloading

For services that need updated credentials without a full restart, use `syringe update` as `ExecReload=`. `syringe update` first re-fetches credentials, then execs into any remaining arguments — allowing you to chain the service's own reload command:

```ini
[Service]
LoadCredential=tls-cert:/run/syringe/syringe.sock
LoadCredential=tls-key:/run/syringe/syringe.sock
ExecReload=/usr/bin/syringe update nginx -s reload
```

Running `systemctl reload nginx.service` will re-fetch the TLS cert and key from syringe, atomically replace them in the service's credentials directory, then exec `nginx -s reload` so nginx picks up the new files. This works by jumping into the service's mount namespace, remounting the credentials directory read-write, and performing the update.

Note: this is a best-effort workaround for [systemd/systemd#21099][systemd-issue-21099]. systemd v260+ will include [`RefreshOnReload=`][systemd-pr-40093] as a native solution.

## Security

- The credential socket is `root:root` mode `0600` — only root (systemd) can connect
- D-Bus method calls are restricted: `Reload` is root-only, `GetSocketPaths` is available to all users
- `syringe update` requires SUID/SGID for credential reloading of non-root services — **no audit has been done, use at your own risk**
- File reads are sandboxed via `sandbox_path` per template

## See also

- [systemd LoadCredential documentation][systemd-load-credential]
- [systemd/systemd#17510 — LoadCredential from AF_UNIX socket][systemd-pr-17510]
- [systemd/systemd#21099 — credential refresh for running services][systemd-issue-21099]
- [systemd/systemd#40093 — RefreshOnReload=][systemd-pr-40093]

[systemd-load-credential]: https://www.freedesktop.org/software/systemd/man/systemd.exec.html#LoadCredential=ID:PATH
[systemd-issue-21099]: https://github.com/systemd/systemd/issues/21099
[systemd-pr-17510]: https://github.com/systemd/systemd/pull/17510
[systemd-pr-40093]: https://github.com/systemd/systemd/pull/40093
[sprig]: https://masterminds.github.io/sprig/
[sockaddr]: https://pkg.go.dev/github.com/hashicorp/go-sockaddr/template
