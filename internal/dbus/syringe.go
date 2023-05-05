package dbus

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
)

func GetServiceSocketPaths(ctx context.Context) (socketPaths map[string]bool, err error) {
	var conn *dbus.Conn

	if conn, err = dbus.ConnectSystemBus(); err != nil {
		err = fmt.Errorf("failed to establish dbus connection: %w", err)
		return
	}
	defer conn.Close()

	var socketPathValue dbus.Variant
	obj := conn.Object(intf, objPath)
	err = obj.CallWithContext(ctx, "ee.zentria.syringe1.Syringe.GetSocketPaths", 0).Store(&socketPathValue)
	if err != nil {
		err = fmt.Errorf("failed to call GetSocketPaths: %w", err)
		return
	}

	socketPathsArray := socketPathValue.Value().([]string)

	socketPaths = make(map[string]bool)
	for _, path := range socketPathsArray {
		socketPaths[path] = true
	}

	return
}

func GetGlobalDebug(ctx context.Context) (debug bool, err error) {
	var conn *dbus.Conn

	if conn, err = dbus.ConnectSystemBus(); err != nil {
		err = fmt.Errorf("failed to establish dbus connection: %w", err)
		return
	}
	defer conn.Close()

	var globalDebugValue dbus.Variant
	obj := conn.Object(intf, objPath)
	err = obj.CallWithContext(ctx, "ee.zentria.syringe1.Syringe.GetGlobalDebug", 0).Store(&globalDebugValue)
	if err != nil {
		err = fmt.Errorf("failed to call GetGlobalDebug: %w", err)
		return
	}

	debug = globalDebugValue.Value().(bool)
	return
}
