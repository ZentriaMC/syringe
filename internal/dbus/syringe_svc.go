package dbus

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"go.uber.org/zap"

	cctx "github.com/ZentriaMC/syringe/internal/ctx"
)

const pkg = `ee.zentria.syringe1`
const intf = pkg + `.Syringe`
const intro = introspect.IntrospectDeclarationString + `
<node xmlns:doc="http://www.freedesktop.org/dbus/1.0/doc.dtd">
    <interface name="` + intf + `">
        <method name="GetSocketPaths">
            <arg direction="out" type="as">
                <doc:doc><doc:summary>Path to the Unix sockets where Syringe is currently listening on</doc:summary></doc:doc>
            </arg>
        </method>
        <method name="Reload">
        	<arg direction="out" type="s">
         		<doc:doc><doc:summary>Reload</doc:summary></doc:doc>
         	</arg>
        </method>
    </interface>
    ` + introspect.IntrospectDataString + `
</node>
`

var objPath = dbus.ObjectPath("/" + strings.ReplaceAll(pkg, ".", "/"))

type syringeService struct {
	ctx context.Context
}

func (s *syringeService) GetSocketPaths() (v []string, err *dbus.Error) {
	v = cctx.SocketPaths(s.ctx)
	return
}

func (s *syringeService) Reload() (result string, err *dbus.Error) {
	return
}

func RegisterSyringeService(ctx context.Context) (err error) {
	var conn *dbus.Conn

	if conn, err = dbus.ConnectSystemBus(); err != nil {
		err = fmt.Errorf("failed to establish dbus connection: %w", err)
		return
	}

	svc := &syringeService{ctx}

	err = conn.Export(svc, objPath, intf)
	if err != nil {
		return
	}

	err = conn.Export(introspect.Introspectable(intro), objPath, "org.freedesktop.DBus.Introspectable")
	if err != nil {
		return
	}

	var registerReply dbus.RequestNameReply
	registerReply, err = conn.RequestName(intf, dbus.NameFlagDoNotQueue)
	if err != nil {
		return
	}

	if registerReply != dbus.RequestNameReplyPrimaryOwner {
		err = fmt.Errorf("failed to register dbus service: %w", errors.New("name already in use"))
		return
	}

	zap.L().Debug("dbus service is running", zap.String("objPath", string(objPath)), zap.String("intf", intf))
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	return
}
