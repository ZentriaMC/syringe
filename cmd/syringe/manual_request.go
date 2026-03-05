package main

import (
	"bytes"
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/ZentriaMC/syringe/internal/request"
)

func manualRequestEntrypoint(_ context.Context, cmd *cli.Command) (err error) {
	unit := cmd.String("unit")
	socket := cmd.String("socket")
	credential := cmd.String("credential")

	var req request.CredentialRequest
	if req, err = request.NewCredentialRequest(unit, credential); err != nil {
		return
	}

	var buf bytes.Buffer
	if _, err = requestCredential(socket, req, &buf); err != nil {
		return
	}

	_, err = os.Stdout.Write(buf.Bytes())
	return
}
