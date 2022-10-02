//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

func withMountNSOf(targetPID int, f func() error) (err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	selfPID := os.Getpid()

	var oldNS int
	oldNSPath := filepath.Join("/proc", fmt.Sprintf("%d", selfPID), "ns/mnt")
	if oldNS, err = syscall.Open(oldNSPath, syscall.O_RDONLY, 0); err != nil {
		err = fmt.Errorf("failed to open current mount namespace %s: %w", oldNSPath, err)
		return
	}

	// Always unshare, even before setns call
	if err = syscall.Unshare(syscall.CLONE_NEWNS); err != nil {
		err = fmt.Errorf("failed to unshare mount namespace: %w", err)
		_ = syscall.Close(oldNS)
		return
	}

	if targetPID > 0 && targetPID != selfPID {
		var newNS int
		newNSPath := filepath.Join("/proc", fmt.Sprintf("%d", targetPID), "ns/mnt")
		if newNS, err = syscall.Open(newNSPath, syscall.O_RDONLY, 0); err != nil {
			err = fmt.Errorf("failed to switch to pid %d mount namespace %s: %w", targetPID, newNSPath, err)
			return
		}

		if err := unix.Setns(newNS, unix.CLONE_NEWNS); err != nil {
			//err = fmt.Errorf("failed to switch to new ns: %w", err)
			zap.L().Panic("failed to switch to new ns", zap.Error(err))
		}

		_ = syscall.Close(newNS)
	}

	defer func() {
		if err := unix.Setns(oldNS, unix.CLONE_NEWNS); err != nil {
			//err = fmt.Errorf("failed to switch back to original ns: %w", err)
			zap.L().Panic("failed to switch back to original ns", zap.Error(err))
		}
		_ = syscall.Close(oldNS)
	}()

	if err = f(); err != nil {
		return
	}

	return
}

func withNewMountNS(f func() error) (err error) {
	return withMountNSOf(0, f)
}

func withRWMount(path string, f func(string) error) (err error) {
	runtime.LockOSThread()
	if err = syscall.Mount("", path, "", syscall.MS_REMOUNT, ""); err != nil {
		err = fmt.Errorf("unable to remount '%s' read-write: %w", path, err)
		return
	}
	defer func() {
		if err := syscall.Mount("", path, "", syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
			zap.L().Warn("unable to re-mount as read only", zap.String("path", path), zap.Error(err))
		}
		runtime.UnlockOSThread()
	}()
	err = f(path)
	return
}
