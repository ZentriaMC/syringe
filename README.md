# Syringe

Credential service implementation, supporting systemd's [LoadCredential][systemd-load-credential] Unix socket feature

Work in progress

## Features

- Load credentials from HashiCorp Vault
    - Has very primitive reading support - no writing support yet.
- Load credentials from file
    - age & PGP support planned.
- Credentials reloading
    - Note that this is a best-effort hack, see [systemd/systemd#21099][systemd-issue-21099]

## Security

This program is using SUID & SGID to reload credentials for non-root user services.

No audit has been done, thus use this feature at your own peril.

## See also
- [systemd/systemd#17510][systemd-pr-17510]

[systemd-load-credential]: https://www.freedesktop.org/software/systemd/man/systemd.exec.html#LoadCredential=ID:PATH
[systemd-issue-21099]: https://github.com/systemd/systemd/issues/21099
[systemd-pr-17510]: https://github.com/systemd/systemd/pull/17510
