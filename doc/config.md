# Configuring Syringe

## Configuration structure

```yaml
---
templates:
  - name: "foo.service"
    credential:
      - "timestamp"
    contents: |
      secret generated @ {{ time "rfc3339" "utc" }}
      
  - name: "bar.service"
    contents: |
      catch-all secret
```

This will provide credential named `timestamp` for `foo.service`, and all credentials for service `bar.service`.

`contents` contains a Go text template string. TODO: document functions.

### Testing configuration

```shell
sudo systemd-run --pipe --collect --unit=foo --property=LoadCredential=timestamp:/run/syringe/syringe.sock /bin/sh -c 'echo $(< "${CREDENTIALS_DIRECTORY}/timestamp")'
```

## Hashicorp Vault integration

It's recommended to run Vault Agent on the machine, which will proxy all Vault API calls without needing to configure
authentication for Syringe.

However this has few security concerns - authentication will be implemented into Syringe directly in near future.

## Integrating Syringe credential reloading into services

Syringe credential reloading logic expects to be called from desired service's `ExecReload`, so you need to override/add
`ExecReload` and prefix the whole command with `syringe update ...`.

### Example (with credentials)

`/etc/systemd/system/nginx.service.d/syringe.conf`:

```ini
[Unit]
ExecReload=
ExecReload=/usr/bin/syringe update /usr/bin/nginx -s reload
LoadCredential=tls_cert_zentria.company:/run/syringe/syringe.sock
LoadCredential=tls_key_zentria.company:/run/syringe/syringe.sock
LoadCredential=tls_fullchain_zentria.company:/run/syringe/syringe.sock
```

### NixOS notes

For NixOS you must use `/run/wrappers/bin/syringe` instead of `${pkgs.syringe}/bin/syringe`.

TODO: expose in module as read-only value.
