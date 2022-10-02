//go:build linux
// +build linux

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/ZentriaMC/syringe/internal/dbus"
	"github.com/ZentriaMC/syringe/internal/request"
)

type PeerCredential = syscall.Ucred

func parseRequest(client *net.UnixConn) (req request.CredentialRequest, peer PeerCredential, err error) {
	var f *os.File
	f, err = client.File()
	if err != nil {
		err = fmt.Errorf("unable to get underlying socket fd")
		return
	}
	defer f.Close()
	fd := int(f.Fd())

	sockAddrRaw, err := syscall.Getpeername(fd)
	if err != nil {
		err = fmt.Errorf("unable to get peer name from fd %d: %w", fd, err)
		return
	}

	sockAddr, ok := sockAddrRaw.(*syscall.SockaddrUnix)
	if !ok {
		err = fmt.Errorf("unable to get peer address: fd %d peername is not type of SockaddrUnix", fd)
		return
	}

	rawRequest := []byte(sockAddr.Name)
	req, err = request.ParseCredentialRequest(rawRequest)
	if err != nil {
		err = fmt.Errorf("unable to parse credential request: %w", err)
		return
	}

	cred, err := syscall.GetsockoptUcred(fd, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil {
		err = fmt.Errorf("unable to get peer credential from fd %d: %w", fd, err)
		return
	}

	peer.Pid = cred.Pid
	peer.Uid = cred.Uid
	peer.Gid = cred.Gid

	name, _, _, err := dbus.GetServiceByPID(context.TODO(), int(cred.Pid))
	if err != nil {
		err = fmt.Errorf("failed to get service name by pid %d: %w", cred.Pid, err)
		return
	}

	if name != req.Unit {
		err = fmt.Errorf("service name mismatch; expected: %s, actual: %s", req.Unit, name)
		return
	}

	return
}
