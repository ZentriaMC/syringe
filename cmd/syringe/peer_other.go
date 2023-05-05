//go:build !linux
// +build !linux

package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/ZentriaMC/syringe/internal/request"
)

type PeerCredential struct {
	Pid int32
	Uid uint32
	Gid uint32
}

func parseRequest(ctx context.Context, client *net.UnixConn) (req request.CredentialRequest, peer PeerCredential, err error) {
	peer.Pid = 0
	peer.Uid = 0
	peer.Gid = 0

	sizeBuf := make([]byte, 4)
	var read int
	if read, err = client.Read(sizeBuf); err != nil {
		return
	} else if read != 4 {
		err = fmt.Errorf("expected %d bytes, got %d\n", 4, read)
		return
	}

	size := binary.BigEndian.Uint32(sizeBuf)
	rawRequest := make([]byte, size)

	if read, err = client.Read(rawRequest); err != nil {
		err = fmt.Errorf("failed to read request: %w", err)
		return
	} else if uint32(read) != size {
		err = fmt.Errorf("expected %d bytes, got %d\n", size, read)
		return
	}

	req, err = request.ParseCredentialRequest(rawRequest)
	if err != nil {
		err = fmt.Errorf("unable to parse credential request: %w", err)
		return
	}

	return
}
