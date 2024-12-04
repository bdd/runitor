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

	dirs := []string{usr.HomeDir}
	globPattern := fmt.Sprintf("/dev/shm/%s_%s*", Name, USER)
	shmDirs, _ := filepath.Glob(globPattern)
	dirs = append(dirs, shmDirs...)

	for _, dir := range dirs {
		if isWritableAndOwned(dir, usr) {
			DIR = dir
			break
		}
	}

	if DIR != "" && isWritableAndOwned(DIR, usr) {
		DIR = filepath.Join(DIR, ".cache", Name)
	} else {
		tempDir, err := os.MkdirTemp("/dev/shm", fmt.Sprintf("%s_%s_XXXXXXXX", Name, USER))
		if err != nil {
			return nil, fmt.Errorf("could not create temporary directory: %w", err)
		}
		DIR = filepath.Join(tempDir, ".cache", Name)
	}

	if err := os.MkdirAll(DIR, 0700); err != nil {
		return nil, fmt.Errorf("could not create cache directory: %w", err)
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
func isWritableAndOwned(path string, usr *user.User) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	if fmt.Sprint(stat.Uid) != usr.Uid {
		return false
	}
	if info.Mode().Perm()&(1<<(uint(7))) == 0 {
		return false
	}
	if !info.IsDir() {
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
