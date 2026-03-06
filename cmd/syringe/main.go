package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap"

	"github.com/ZentriaMC/syringe/internal/version"
)

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)

	cmd := &cli.Command{
		Name:    "syringe",
		Usage:   "systemd LoadCredential service implementation",
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Whether to enable debug logging",
				Value:   false,
				Sources: cli.EnvVars("SYRINGE_DEBUG"),
			},
			&cli.BoolFlag{
				Name:    "debug-global",
				Usage:   "Whether to enable debug logging for all syringe clients via D-Bus",
				Value:   false,
				Sources: cli.EnvVars("SYRINGE_DEBUG_GLOBAL"),
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "server",
				Usage:  "Run credentials server",
				Action: serverEntrypoint,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:      "config",
						Usage:     "Path to configuration file",
						Value:     "/etc/syringe/config.yml",
						TakesFile: true,
						Sources:   cli.EnvVars("SYRINGE_SERVER_CONFIG"),
					},
					&cli.StringFlag{
						Name:    "socket",
						Usage:   "Path to listen socket. Unnecessary when using systemd socket activation",
						Value:   "/tmp/syringe.sock",
						Sources: cli.EnvVars("SYRINGE_SERVER_SOCKET"),
					},
					&cli.BoolFlag{
						Name:    "dbus",
						Usage:   "Whether to enable dbus support",
						Value:   runtime.GOOS == "linux",
						Sources: cli.EnvVars("SYRINGE_SERVER_DBUS"),
					},
				},
			},
			{
				Name:   "request",
				Usage:  "Request a credential manually",
				Action: manualRequestEntrypoint,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "socket",
						Usage: "Syringe socket path",
						Value: "/tmp/syringe.sock",
					},
					&cli.StringFlag{
						Name:     "unit",
						Usage:    "Unit name to request credential for",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "credential",
						Usage:    "Credential name to request",
						Required: true,
					},
				},
			},
			{
				Name:   "update",
				Usage:  "Update credentials for a service. Meant to be used inside service ExecReload",
				Action: updateEntrypoint,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			err := setupLogging(cmd.Bool("debug") || cmd.Bool("debug-global"))
			return ctx, err
		},
		After: func(ctx context.Context, cmd *cli.Command) error {
			_ = zap.L().Sync()
			return nil
		},
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		zap.L().Error("unhandled error", zap.Error(err))
		os.Exit(1)
	}
}
