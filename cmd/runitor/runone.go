// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

// createLockFile creates a lock file for the given command
func createLockFile(args []string) (*os.File, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("could not get current user: %w", err)
	}
	USER := usr.Username

	var DIR string

	dirs := []string{
		filepath.Join("/dev/shm", Name+"_"+USER),
		filepath.Join(os.TempDir(), Name+"_"+USER),
		filepath.Join(usr.HomeDir, ".cache"),
	}

	for _, dir := range dirs {
		if isWritableAndOwned(dir) {
			DIR = dir
			break
		}
	}

	if DIR == "" {
		return nil, fmt.Errorf("could not find a writable cache directory")
	}

	cmd := strings.Join(args, " ")
	cmdHash := md5Hash(cmd)
	flag := filepath.Join(DIR, cmdHash)

	file, err := os.OpenFile(flag, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("could not open flag file: %w", err)
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()
		if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
			return nil, nil
		}
		return nil, fmt.Errorf("could not lock flag file: %w", err)
	}

	return file, nil
}

// isWritableAndOwned checks if the path is writable and owned by the user
func isWritableAndOwned(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0700); err != nil {
			return false
		}
		info, err = os.Stat(path)
		if err != nil {
			return false
		}
	} else if err != nil {
		return false
	}

	if !info.IsDir() {
		return false
	}

	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		return false
	}

	return true
}

// md5Hash returns the md5 hash of the given text
func md5Hash(text string) string {
	hasher := md5.New()
	io.WriteString(hasher, text)
	return hex.EncodeToString(hasher.Sum(nil))
}

// unlockAndRemove unlocks the file and removes it
func unlockAndRemove(file *os.File) {
	if file == nil {
		return
	}
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	file.Close()
	os.Remove(file.Name())
}

// setupSignalHandler sets up a signal handler for the given lock file
func setupSignalHandler(lockFile *os.File) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-c
		unlockAndRemove(lockFile)
		os.Exit(1)
	}()
}
