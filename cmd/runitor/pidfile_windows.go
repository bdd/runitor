// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD

//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// errSharingViolation is Windows' ERROR_SHARING_VIOLATION, not exported by
// the stdlib syscall package.
const errSharingViolation syscall.Errno = 32

// openAndLockPidfile opens path with exclusive write share (readers are
// still allowed so operators can inspect the PID file while runitor runs).
// Another runitor trying to open the same path receives
// ERROR_SHARING_VIOLATION which we translate to errPidfileBusy.
//
// Unlike the Unix path, the file is NOT removed automatically on crash. A
// follow-up runitor re-opens and overwrites the stale PID, matching the
// classic Unix pidfile semantics.
func openAndLockPidfile(path string) (*os.File, error) {
	u16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("invalid pidfile path %q: %w", path, err)
	}
	h, err := syscall.CreateFile(
		u16,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ,
		nil,
		syscall.OPEN_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		if errors.Is(err, errSharingViolation) {
			return nil, errPidfileBusy
		}
		return nil, fmt.Errorf("could not open pidfile: %w", err)
	}
	return os.NewFile(uintptr(h), path), nil
}

// releasePidfile closes the handle and removes the file.
func releasePidfile(f *os.File) {
	path := f.Name()
	f.Close()
	os.Remove(path)
}
