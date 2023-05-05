//go:build !linux
// +build !linux

package main

import (
	"context"
	"errors"
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

	err = errors.New("not implemented")
	return
}
