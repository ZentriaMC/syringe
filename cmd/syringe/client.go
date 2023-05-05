package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"runtime"

	"github.com/ZentriaMC/syringe/internal/request"
	"github.com/ZentriaMC/syringe/internal/secret"
)

func requestCredential(socket string, cr request.CredentialRequest, out io.Writer) (n int64, err error) {
	var conn *net.UnixConn
	var laddr, raddr *net.UnixAddr
	var request []byte

	if raddr, err = net.ResolveUnixAddr("unix", socket); err != nil {
		return
	}

	if request, err = cr.Bytes(); err != nil {
		err = fmt.Errorf("failed to serialize the credential request: %w", err)
		return
	}

	if runtime.GOOS == "linux" {
		laddr = &net.UnixAddr{
			Net:  raddr.Net,
			Name: string(request),
		}
	}

	if conn, err = net.DialUnix("unix", laddr, raddr); err != nil {
		err = fmt.Errorf("failed to connect to '%s': %w", socket, err)
		return
	}

	defer func() { _ = conn.Close() }()

	if runtime.GOOS != "linux" {
		if _, err = conn.Write(binary.BigEndian.AppendUint32(nil, uint32(len(request)))); err != nil {
			err = fmt.Errorf("failed to write credential request: %w", err)
			return
		}
		if _, err = conn.Write(request); err != nil {
			err = fmt.Errorf("failed to write credential request: %w", err)
			return
		}
	}

	if n, err = io.Copy(out, io.LimitReader(conn, secret.MAX_CREDENTIAL_SIZE)); err != nil {
		err = fmt.Errorf("failed to read credential request response: %w", err)
		return
	}
	return
}
