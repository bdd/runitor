// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD

//go:build unix

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// openAndLockPidfile opens path (creating it if needed) and takes a
// non-blocking exclusive flock. The flock is automatically released by the
// kernel if runitor dies, so a stale file left on disk doesn't permanently
// block future invocations.
func openAndLockPidfile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open pidfile: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errPidfileBusy
		}
		return nil, fmt.Errorf("could not lock pidfile: %w", err)
	}
	return f, nil
}

// releasePidfile removes the file before dropping the flock so that a new
// runitor opening the same path right after us creates a fresh inode
// instead of racing on our about-to-be-unlinked one.
func releasePidfile(f *os.File) {
	path := f.Name()
	os.Remove(path)
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	f.Close()
}
