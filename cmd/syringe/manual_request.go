package main

import (
	"bytes"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ZentriaMC/syringe/internal/request"
)

func manualRequestEntrypoint(clictx *cli.Context) (err error) {
	unit := clictx.String("unit")
	socket := clictx.String("socket")
	credential := clictx.String("credential")

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
