// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAcquirePidfileWritesOurPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runitor.pid")
	pf, err := acquirePidfile(path)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer pf.Release()

	if got := readPidfile(path); got != os.Getpid() {
		t.Errorf("pidfile contains %d, expected %d", got, os.Getpid())
	}
}

func TestAcquirePidfileBusy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runitor.pid")
	first, err := acquirePidfile(path)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer first.Release()

	second, err := acquirePidfile(path)
	if !errors.Is(err, errPidfileBusy) {
		t.Errorf("second acquire: expected errPidfileBusy, got (%v, %v)", second, err)
	}
	if second != nil {
		second.Release()
	}
}

func TestPidfileReleaseRemovesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runitor.pid")
	pf, err := acquirePidfile(path)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	pf.Release()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected pidfile removed, stat err = %v", err)
	}
}

func TestPidfileReleaseIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runitor.pid")
	pf, err := acquirePidfile(path)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	pf.Release()
	pf.Release() // must not panic
}

func TestAcquireAfterStalePidfile(t *testing.T) {
	// Simulate a pidfile left on disk by a crashed runitor: the file
	// exists with a stale PID but no live process holds the OS lock.
	path := filepath.Join(t.TempDir(), "runitor.pid")
	if err := os.WriteFile(path, []byte("99999\n"), 0644); err != nil {
		t.Fatalf("seed stale pidfile: %v", err)
	}

	pf, err := acquirePidfile(path)
	if err != nil {
		t.Fatalf("acquire after stale: %v", err)
	}
	defer pf.Release()

	if got := readPidfile(path); got != os.Getpid() {
		t.Errorf("stale PID not overwritten; got %d, expected %d", got, os.Getpid())
	}
}

func TestReadPidfile(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name    string
		content string
		seed    bool
		want    int
	}{
		{"missing", "", false, 0},
		{"empty", "", true, 0},
		{"whitespace", "   \n", true, 0},
		{"garbage", "not a number", true, 0},
		{"negative", "-1", true, 0},
		{"zero", "0", true, 0},
		{"valid", "12345", true, 12345},
		{"trailing newline", "67890\n", true, 67890},
		{"leading whitespace", "  42\n", true, 42},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(c.name, " ", "_")+".pid")
			if c.seed {
				if err := os.WriteFile(path, []byte(c.content), 0644); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}
			if got := readPidfile(path); got != c.want {
				t.Errorf("readPidfile(%q)=%d, want %d", c.content, got, c.want)
			}
		})
	}
}

func TestWritePIDFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runitor.pid")
	pf, err := acquirePidfile(path)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer pf.Release()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := strings.TrimSpace(string(raw))
	if _, err := strconv.Atoi(s); err != nil {
		t.Errorf("pidfile content %q is not a decimal integer: %v", s, err)
	}
}
