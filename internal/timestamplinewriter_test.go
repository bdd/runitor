// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package internal_test

import (
	"strings"
	"testing"
	"time"

	. "bdd.fi/x/runitor/internal"
)

func TestTimestampLineWriterPrefixesEachLine(t *testing.T) {
	var out strings.Builder
	w := NewTimestampLineWriter(&out)
	now := time.Now()
	w.Now = func() time.Time { return now }

	const input = "first\nsecond\n"
	n, err := w.Write([]byte(input))
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("short write: got %d, want %d", n, len(input))
	}

	prefix := now.Format(w.Format) + " "
	want := prefix + "first\n" + prefix + "second\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected output:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestTimestampLineWriterHandlesChunkedWrites(t *testing.T) {
	var out strings.Builder
	w := NewTimestampLineWriter(&out)
	now := time.Now()
	w.Now = func() time.Time { return now }

	chunks := []string{"hel", "lo\nwo", "rld", "\n"}
	for _, chunk := range chunks {
		n, err := w.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if n != len(chunk) {
			t.Fatalf("short write: got %d, want %d", n, len(chunk))
		}
	}

	prefix := now.Format(w.Format) + " "
	want := prefix + "hello\n" + prefix + "world\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected output:\nwant: %q\ngot:  %q", want, got)
	}
}
