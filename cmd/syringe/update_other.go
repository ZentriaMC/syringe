//go:build !linux

package main

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"
)

func updateEntrypoint(_ context.Context, _ *cli.Command) (err error) {
	err = errors.New("not implemented")
	return
}
