//go:build linux
// +build linux

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/ZentriaMC/syringe/internal/dbus"
	"github.com/ZentriaMC/syringe/internal/request"
	"github.com/ZentriaMC/syringe/internal/secret"
)

const (
	CredentialsDir = "/run/credentials"
)

func updateEntrypoint(clictx *cli.Context) (err error) {
	runtime.GOMAXPROCS(1)
	ctx := clictx.Context

	if os.Getuid() != 0 && os.Geteuid() != 0 {
		zap.L().Error("effective uid is not 0, very likely unable to update credentials", zap.Int("uid", os.Getuid()), zap.Int("euid", os.Geteuid()))
	}

	// Get unit where we're being called from
	unitName, mainPID, managedCredentials, err := dbus.GetServiceByPID(ctx, os.Getpid())
	if err != nil {
		err = fmt.Errorf("failed to determine unit from context (PID: %d): %w", os.Getpid(), err)
		return
	}

	// Do namespace jumping, because ExecReload does not allow access to credentials
	// self -> unshare -> change ns -> unshare again to not affect real service ns
	err = withMountNSOf(mainPID, func() (err error) {
		credsPath := filepath.Join(CredentialsDir, unitName)
		if _, err = os.Stat(credsPath); os.IsNotExist(err) {
			err = fmt.Errorf("credentials path '%s' does not exist: %w", credsPath, err)
			return
		}

		return withNewMountNS(func() error {
			return withRWMount(credsPath, func(credsPath string) error {
				return processCredentialsDir(credsPath, unitName, managedCredentials)
			})
		})
	})
	if err != nil {
		err = fmt.Errorf("failed to update credentials: %w", err)
		return
	}

	if len(os.Args) <= 2 {
		zap.L().Debug("nothing to execute, exiting", zap.Error(err))
		return
	}

	if err = dropPermissions(); err != nil {
		err = fmt.Errorf("failed to drop permissions: %w", err)
		return
	}

	argv := os.Args[2:]
	if err = unix.Exec(argv[0], argv, os.Environ()); err != nil {
		err = fmt.Errorf("failed to exec: %w", err)
		return
	}

	return
}

func processCredentialsDir(credsPath string, unitName string, managedCredentials map[string]string) (err error) {
	var existingSecrets []os.DirEntry
	if existingSecrets, err = os.ReadDir(credsPath); err != nil {
		err = fmt.Errorf("failed to list secrets in %s: %w", credsPath, err)
		return
	}

	type updatedSecret struct {
		Path         string
		Replacement  string
		WrittenBytes int64
	}
	var updatedSecrets []updatedSecret
	for _, secretFileEntry := range existingSecrets {
		credential := secretFileEntry.Name()
		var credentialSocket string
		var ok bool

		if credentialSocket, ok = managedCredentials[credential]; !ok {
			continue
		}

		// Check if socket is absolute path, and actually a socket
		if st, err := os.Stat(credentialSocket); os.IsNotExist(err) || !filepath.IsAbs(credentialSocket) || st.Mode()&os.ModeSocket == 0 {
			continue
		}

		var us updatedSecret
		us.Path, us.Replacement, err = createReplacementSecret(credsPath, credential, func(w io.Writer) (err error) {
			var cr request.CredentialRequest
			if cr, err = request.NewCredentialRequest(unitName, credential); err != nil {
				err = fmt.Errorf("failed to create credential request: %w", err)
				return
			}

			if us.WrittenBytes, err = requestCredential(credentialSocket, cr, w); err != nil {
				err = fmt.Errorf("failed to create replacement secret '%s': %w", credential, err)
			}

			return
		})

		updatedSecrets = append(updatedSecrets, us)
	}

	// Replace all secrets more or less atomically
	for _, update := range updatedSecrets {
		// If update is empty, do not bother replacing it
		if update.WrittenBytes == 0 {
			zap.L().Debug("skipping empty replacement credential", zap.String("path", update.Replacement))
			_ = os.Remove(update.Replacement)
			continue
		}

		if err = os.Rename(update.Replacement, update.Path); err != nil {
			err = fmt.Errorf("failed to replace '%s' with '%s': %w", update.Replacement, update.Path, err)
			return
		}
	}
	return
}

func createReplacementSecret(credsDir, credential string, f func(io.Writer) error) (original, replacement string, err error) {
	var stat os.FileInfo
	original = filepath.Join(credsDir, credential)
	replacement = filepath.Join(credsDir, fmt.Sprintf(".%s.%d", credential, time.Now().Unix()))

	if stat, err = os.Stat(original); err != nil {
		err = fmt.Errorf("failed to stat secret '%s': %w", credential, err)
		return
	}

	// Grab original secret's owner
	var uid, gid int
	if stat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = int(stat.Uid)
		gid = int(stat.Gid)
	} else {
		uid = os.Getuid()
		gid = os.Getgid()
	}

	var replacementFile *os.File
	if replacementFile, err = os.OpenFile(replacement, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, stat.Mode()); err != nil {
		err = fmt.Errorf("failed to open replacement secret '%s': %w", credential, err)
		return
	}

	defer func() { _ = replacementFile.Close() }()

	if err = f(replacementFile); err != nil {
		err = fmt.Errorf("failed to write replacement secret '%s': %w", credential, err)
		return
	}

	// Ensure that replacement credentials have same owner info
	if err = os.Chown(replacement, uid, gid); err != nil {
		err = fmt.Errorf("failed to chown replacement secret '%s': %w", credential, err)
		return
	}

	return
}
