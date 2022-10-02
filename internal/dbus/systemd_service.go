package dbus

import (
	"context"
	"fmt"
	"os"
	"strings"

	sdunit "github.com/coreos/go-systemd/v22/unit"
	"github.com/godbus/dbus/v5"
	"go.uber.org/zap"
)

func GetServiceByPID(ctx context.Context, pid int) (name string, mainPID int, managedCredentials map[string]string, err error) {
	var syringeSocketPaths map[string]bool
	if syringeSocketPaths, err = GetServiceSocketPaths(ctx); err != nil {
		return
	}

	var conn *dbus.Conn

	if conn, err = dbus.ConnectSystemBus(); err != nil {
		err = fmt.Errorf("failed to establish dbus connection: %w", err)
		return
	}
	defer conn.Close()

	managedCredentials = make(map[string]string)

	// Grab unit
	var unit dbus.ObjectPath
	obj := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
	err = obj.CallWithContext(ctx, "org.freedesktop.systemd1.Manager.GetUnitByPID", 0, uint32(pid)).Store(&unit)
	if err != nil {
		err = fmt.Errorf("failed to call GetUnitByPid: %w", err)
		return
	}

	// Read unit name
	var nameValue dbus.Variant
	obj = conn.Object("org.freedesktop.systemd1", unit)
	if nameValue, err = obj.GetProperty("org.freedesktop.systemd1.Unit.Id"); err != nil {
		err = fmt.Errorf("failed to read unit Id property: %w", err)
		return
	}

	// NOTE: this is always string
	name = nameValue.Value().(string)

	// Read service main PID
	var mainPIDValue dbus.Variant
	obj = conn.Object("org.freedesktop.systemd1", unit)
	if mainPIDValue, err = obj.GetProperty("org.freedesktop.systemd1.Service.MainPID"); err != nil {
		err = fmt.Errorf("failed to read unit MainPID property: %w", err)
		return
	}

	mainPID = int(mainPIDValue.Value().(uint32))

	// Grab fragment path
	var fragmentPathValue dbus.Variant
	obj = conn.Object("org.freedesktop.systemd1", unit)
	if fragmentPathValue, err = obj.GetProperty("org.freedesktop.systemd1.Unit.FragmentPath"); err != nil {
		err = fmt.Errorf("failed to read unit FragmentPath property: %w", err)
		return
	}

	var fragmentFile *os.File
	var sections []*sdunit.UnitSection
	fragmentPath := fragmentPathValue.Value().(string)
	if fragmentFile, err = os.Open(fragmentPath); err != nil {
		err = fmt.Errorf("failed to open unit fragment '%s': %w", fragmentPath, err)
		return
	}
	defer func() { _ = fragmentFile.Close() }()

	if sections, err = sdunit.DeserializeSections(fragmentFile); err != nil {
		err = fmt.Errorf("failed to parse unit fragment '%s': %w", fragmentPath, err)
		return
	}

	for _, section := range sections {
		if section.Section != "Service" {
			continue
		}

		for _, entry := range section.Entries {
			if entry.Name != "LoadCredential" && entry.Name != "LoadCredentialEncrypted" {
				continue
			}

			split := strings.SplitN(entry.Value, ":", 2)
			credential := split[0]
			path := split[1]

			// TODO: Check if files are actually the same
			_, isSyringeSocket := syringeSocketPaths[path]
			if !isSyringeSocket {
				zap.L().Debug("skipping credential, not managed by syringe", zap.String("unit", name), zap.String("credential", credential), zap.String("path", path))
				continue
			}

			managedCredentials[credential] = path
		}
	}

	return
}
