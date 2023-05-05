package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/v22/activation"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/ZentriaMC/syringe/internal/config"
	cctx "github.com/ZentriaMC/syringe/internal/ctx"
	"github.com/ZentriaMC/syringe/internal/dbus"
	"github.com/ZentriaMC/syringe/internal/secret"
	"github.com/ZentriaMC/syringe/internal/templatemap"
)

func serverEntrypoint(clictx *cli.Context) (err error) {
	// Drop possible elevated permissions
	if err = dropPermissions(); err != nil {
		err = fmt.Errorf("failed to drop permissions: %w", err)
		return
	}

	var socketPaths []string
	var unixListeners []*net.UnixListener
	var listeners []net.Listener
	if listeners, err = activation.Listeners(); err != nil {
		return
	}

	unixListeners, socketPaths = filterUnixListeners(listeners)
	if len(unixListeners) == 0 {
		socketPath := clictx.String("socket")
		var server *net.UnixListener
		var addr *net.UnixAddr

		if socketPath, err = filepath.Abs(socketPath); err != nil {
			err = fmt.Errorf("failed to parse address: %w", err)
			return
		}

		if addr, err = net.ResolveUnixAddr("unix", socketPath); err != nil {
			err = fmt.Errorf("failed to parse address: %w", err)
			return
		}

		if server, err = net.ListenUnix("unix", addr); err != nil {
			err = fmt.Errorf("failed to listen: %w", err)
			return
		}

		server.SetUnlinkOnClose(true)
		defer func() { _ = server.Close() }()

		unixListeners = append(unixListeners, server)
		socketPaths = append(socketPaths, socketPath)
	}

	// Load credential templates
	configFile := clictx.Path("config")
	var cfg *config.Config
	if cfg, err = config.LoadConfig(configFile); err != nil {
		err = fmt.Errorf("unable to load configuration from '%s': %w", configFile, err)
		return
	}

	tm := templatemap.NewTemplateMap()
	if err = tm.Populate(cfg); err != nil {
		return
	}

	// Connect to the vault
	var vault *vaultapi.Client
	vaultConfig := vaultapi.DefaultConfig()
	if vault, err = vaultapi.NewClient(vaultConfig); err != nil {
		err = fmt.Errorf("failed to connect to the vault: %w", err)
		return
	}

	ctx := cctx.Apply(
		clictx.Context,
		cctx.WithVaultClient(vault),
		cctx.WithTemplateMap(tm),
		cctx.WithSocketPaths(socketPaths),
	)

	if clictx.Bool("dbus") {
		if err = dbus.RegisterSyringeService(ctx); err != nil {
			err = fmt.Errorf("failed to register dbus service: %w", err)
			return
		}
	}

	for idx, listener := range unixListeners {
		socketPath := socketPaths[idx]
		zap.L().Debug("listening", zap.String("on", socketPath))

		go func(server *net.UnixListener, socketPath string) {
			for {
				client, err := server.AcceptUnix()
				if err != nil {
					zap.L().Warn("failed to accept connection", zap.Error(err))
					continue
				}

				if err := client.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
					zap.L().Warn("failed to set client rw deadline", zap.Error(err))
				}

				go serve(ctx, client)
			}
		}(listener, socketPath)
	}

	<-ctx.Done()
	return
}

func dropPermissions() (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if os.Getgid() != os.Getegid() {
		if err = syscall.Setgid(os.Getgid()); err != nil {
			err = fmt.Errorf("failed to setgid: %w", err)
			return
		}
	}

	if os.Getuid() != os.Geteuid() {
		if err = syscall.Setuid(os.Getuid()); err != nil {
			err = fmt.Errorf("failed to setuid: %w", err)
			return
		}
	}

	return
}

func filterUnixListeners(listeners []net.Listener) (unixListeners []*net.UnixListener, socketPaths []string) {
	for idx, listener := range listeners {
		unixListener, ok := listener.(*net.UnixListener)
		if !ok {
			zap.L().Debug("skipping listener - not unix listener", zap.Int("idx", idx), zap.String("type", listener.Addr().Network()))
			continue
		}

		socketPath := unixListener.Addr().(*net.UnixAddr).Name

		unixListeners = append(unixListeners, unixListener)
		socketPaths = append(socketPaths, socketPath)
	}
	return
}

func serve(ctx context.Context, client *net.UnixConn) {
	defer client.Close()

	req, peer, err := parseRequest(ctx, client)
	if err != nil {
		zap.L().Error("failed to process credential request", zap.Error(err))
		return
	}

	zap.L().Debug("credential request",
		zap.String("unit", req.Unit),
		zap.String("credential", req.Credential),
		zap.Int32("pid", peer.Pid),
		zap.Uint32("uid", peer.Uid),
		zap.Uint32("gid", peer.Gid),
	)

	ctx = cctx.Apply(ctx, cctx.WithCredentialRequest(req.Unit, req.Credential))
	if err = secret.Render(ctx, client); err != nil {
		zap.L().Error("failed to respond with credential", zap.Error(err))
	}
}
