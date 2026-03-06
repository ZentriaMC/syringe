//go:build linux

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"golang.org/x/sys/unix"
)

const syringeUpdateBin = "/usr/bin/syringe-update"

func updateEntrypoint(ctx context.Context, cmd *cli.Command) (err error) {
	argv := []string{syringeUpdateBin}

	if cmd.Bool("debug") {
		argv = append(argv, "--debug")
	}
	if cmd.Bool("debug-global") {
		argv = append(argv, "--debug-global")
	}

	// Pass through any trailing arguments (e.g. nginx -s reload)
	argv = append(argv, cmd.Args().Slice()...)

	if err = unix.Exec(syringeUpdateBin, argv, os.Environ()); err != nil {
		err = fmt.Errorf("failed to exec %s: %w", syringeUpdateBin, err)
	}
	return
}
