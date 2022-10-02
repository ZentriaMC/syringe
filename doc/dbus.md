# Syringe D-BUS API

TODO: auto-generate documentation somehow.

## ee.zentria.syringe1.Syringe.GetSocketPaths

Returns a list of sockets where Syringe server is listening on.

Takes no arguments, returns string array.

```shell
dbus-send --system --print-reply --type=method_call --dest=ee.zentria.syringe1.Syringe /ee/zentria/syringe1 ee.zentria.syringe1.Syringe.GetSocketPaths
```
