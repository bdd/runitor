// Copyright 2020 - 2022, Berk D. Demir and the runitor contributors
// SPDX-License-Identifier: 0BSD
package internal_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	. "bdd.fi/x/runitor/internal"
)

const RCap = 8

func TestRead(t *testing.T) {
	tests := map[string]struct {
		str string
		out string
	}{
		"empty":     {str: "", out: ""},
		"half full": {str: "0123", out: "0123"},
		"full":      {str: "01234567", out: "01234567"},
		"wrapped":   {str: "0123456789", out: "23456789"},
	}

	for name, tc := range tests {
		rb := NewRingBuffer(RCap)
		fmt.Fprint(rb, tc.str)
		out, err := io.ReadAll(rb)
		if err != nil {
			t.Errorf("%s: read failed: %v", name, err)
		}
		outstr := string(out)
		if outstr != tc.out {
			t.Errorf("%s: expected to read '%s', got '%s'", name, tc.out, outstr)
		}
	}
}

func TestNoWriteAfterRead(t *testing.T) {
	rb := NewRingBuffer(RCap)
	rb.Write([]byte{1})
	io.ReadAll(rb)

	if _, err := rb.Write([]byte{2}); err == nil || !errors.Is(err, ErrReadOnly) {
		t.Errorf("expected ring buffer to become read only after first read and receive ErrReadOnly but got err '%v'", err)
	}
}

func TestWriteAllocs(t *testing.T) {
	rb := NewRingBuffer(RCap)
	tb := make([]byte, RCap+1)
	allocs := testing.AllocsPerRun(1, func() {
		rb.Write(tb)
	})

	if allocs != 0 {
		t.Errorf("expected 0 allocations, observed %f\n", allocs)
	}
}

func TestReadAllocs(t *testing.T) {
	rb := NewRingBuffer(RCap)
	rb.Write(make([]byte, RCap+1))
	p := make([]byte, RCap)

	allocs := testing.AllocsPerRun(1, func() {
		rb.Read(p)
	})

	if allocs != 0 {
		t.Errorf("expected 0 allocations, observed %f\n", allocs)
	}
}
