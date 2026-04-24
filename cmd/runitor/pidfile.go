// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// errPidfileBusy signals that another live runitor already holds the file.
var errPidfileBusy = errors.New("pidfile is held by another process")

// Pidfile is an acquired exclusive hold on a user-provided path with our PID
// recorded inside. Closing it releases the hold and removes the file.
type Pidfile struct {
	file *os.File
	once sync.Once
}

// Release drops the hold and cleans up the file. Safe to call multiple times
// and concurrently; only the first invocation actually releases.
func (p *Pidfile) Release() {
	if p == nil || p.file == nil {
		return
	}
	p.once.Do(func() { releasePidfile(p.file) })
}

// acquirePidfile opens path, takes an exclusive OS-level lock on it, and
// records the current PID. If another live runitor already holds the file,
// returns errPidfileBusy. A stale file left behind by a crashed runitor is
// commandeered: the OS lock is re-taken and the PID is overwritten.
func acquirePidfile(path string) (*Pidfile, error) {
	f, err := openAndLockPidfile(path)
	if err != nil {
		return nil, err
	}
	if err := writePID(f, os.Getpid()); err != nil {
		f.Close()
		return nil, fmt.Errorf("could not write pidfile: %w", err)
	}
	return &Pidfile{file: f}, nil
}

// readPidfile reads and parses the PID from path. Returns 0 if the file is
// missing, empty, or malformed.
func readPidfile(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

// writePID truncates f and writes pid followed by a newline.
func writePID(f *os.File, pid int) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := fmt.Fprintf(f, "%d\n", pid)
	return err
}

// setupSignalHandler arranges for the pidfile to be released on termination
// signals. Installing the handler causes Go to bypass its default terminate-
// on-signal behavior, so we exit explicitly with 128+signum to match shell
// conventions (SIGINT → 130, SIGTERM → 143, SIGHUP → 129).
func setupSignalHandler(pf *Pidfile) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		sig := <-c
		pf.Release()
		if s, ok := sig.(syscall.Signal); ok {
			os.Exit(128 + int(s))
		}
		os.Exit(1)
	}()
}
