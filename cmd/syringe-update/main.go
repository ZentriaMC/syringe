package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/urfave/cli/v3"
	"go.uber.org/zap"

	slog "github.com/ZentriaMC/syringe/internal/log"
	"github.com/ZentriaMC/syringe/internal/version"
)

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)

	cmd := &cli.Command{
		Name:    "syringe-update",
		Usage:   "Update credentials for a service. Meant to be used inside service ExecReload",
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
		Action: updateEntrypoint,
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			err := slog.SetupLogging(cmd.Bool("debug") || cmd.Bool("debug-global"))
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
