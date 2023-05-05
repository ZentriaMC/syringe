package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/ZentriaMC/syringe/internal/version"
)

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)

	app := &cli.App{
		Name:    "syringe",
		Usage:   "systemd LoadCredential service implementation",
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Whether to enable debug logging",
				Value: false,
				EnvVars: []string{
					"SYRINGE_DEBUG",
				},
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "server",
				Usage:  "Run credentials server",
				Action: serverEntrypoint,
				Flags: []cli.Flag{
					&cli.PathFlag{
						Name:  "config",
						Usage: "Path to configuration file",
						Value: "/etc/syringe/config.yml",
						EnvVars: []string{
							"SYRINGE_SERVER_CONFIG",
						},
					},
					&cli.StringFlag{
						Name:  "socket",
						Usage: "Path to listen socket. Unnecessary when using systemd socket activation",
						Value: "/tmp/syringe.sock",
						EnvVars: []string{
							"SYRINGE_SERVER_SOCKET",
						},
					},
					&cli.BoolFlag{
						Name:  "dbus",
						Usage: "Whether to enable dbus support",
						Value: runtime.GOOS == "linux",
						EnvVars: []string{
							"SYRINGE_SERVER_DBUS",
						},
					},
				},
			},
			{
				Name:   "update",
				Usage:  "Update credentials for a service. Meant to be used inside service ExecReload",
				Action: updateEntrypoint,
			},
		},
		Before: func(cctx *cli.Context) (err error) {
			if err = setupLogging(cctx.Bool("debug")); err != nil {
				return
			}

			return
		},
		After: func(ctx *cli.Context) (err error) {
			_ = zap.L().Sync()
			return
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		zap.L().Error("unhandled error", zap.Error(err))
		os.Exit(1)
	}
}
