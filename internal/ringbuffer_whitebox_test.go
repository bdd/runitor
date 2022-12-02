// Copyright 2020 - 2022, Berk D. Demir and the runitor contributors
// SPDX-License-Identifier: 0BSD
package internal

import (
	"fmt"
	"testing"
)

func TestWriteWhitebox(t *testing.T) {
	const RCap = 8

	rb := NewRingBuffer(RCap)
	if rbcap := rb.Cap(); rbcap != RCap {
		t.Errorf("expected Cap() to return %d but got %d", RCap, rbcap)
	}

	tests := []struct {
		name string
		str  string
		buf  string
	}{
		{name: "simple write", str: "abc", buf: "abc"},
		{name: "wrap", str: "012345", buf: "5bc01234"},
		{name: "overrun discard", str: "0123456789", buf: "78923456"},
		{name: "zero byte write", str: "", buf: "78923456"},
	}

	lenExp := 0

	for _, tc := range tests {
		n, err := fmt.Fprint(rb, tc.str)
		if err != nil {
			t.Errorf("%s: expected Write to succeed, got err '%v'", tc.name, err)
		}

		if n != len(tc.str) {
			t.Errorf("%s: expected Write to return %d, got %d", tc.name, len(tc.str), n)
		}

		lenExp = (lenExp + n)
		if lenExp > rb.Cap() {
			lenExp = rb.Cap()
		}

		if rblen := rb.Len(); rblen != lenExp {
			t.Errorf("%s: expected Len to return %d, got %d", tc.name, lenExp, rblen)
		}

		if tc.buf != string(rb.buf) {
			t.Errorf("%s: expected ring buffer to be '%s', got '%s'", tc.name, tc.buf, string(rb.buf))
		}
	}
}
