//go:build !linux
// +build !linux

package main

import (
	"errors"

	"github.com/urfave/cli/v2"
)

func updateEntrypoint(clictx *cli.Context) (err error) {
	err = errors.New("not implemented")
	return
}
