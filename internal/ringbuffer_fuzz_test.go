// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package internal_test

import (
	"bytes"
	"io"
	"testing"

	. "bdd.fi/x/runitor/internal"
)

func FuzzRingBuffer(f *testing.F) {
	// Seed corpus
	f.Add([]byte{10, 5, 'h', 'e', 'l', 'l', 'o'}) // cap 10, chunk 5, "hello"
	f.Add([]byte{5, 2, 'w', 'o', 'r', 'l', 'd'})  // cap 5, chunk 2, "world"
	f.Add([]byte{4, 1, '1', '2', '3', '4', '5', '6'}) // cap 4, chunk 1, wrap behavior

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 2 {
			return
		}

		// Use first byte for capacity (avoid 0)
		capacity := int(data[0])
		if capacity == 0 {
			capacity = 1
		}

		// Use second byte for chunk size (avoid 0)
		chunkSize := int(data[1])
		if chunkSize == 0 {
			chunkSize = 1
		}

		// Remaining data is the payload to write
		payload := data[2:]
		rb := NewRingBuffer(capacity)

		// Calculate expected result: the last 'capacity' bytes of payload
		var expected []byte
		if len(payload) > capacity {
			expected = payload[len(payload)-capacity:]
		} else {
			expected = payload
		}

		// Write payload in chunks to verify partial writes and wrapping
		for i := 0; i < len(payload); i += chunkSize {
			end := i + chunkSize
			if end > len(payload) {
				end = len(payload)
			}
			chunk := payload[i:end]

			n, err := rb.Write(chunk)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			if n != len(chunk) {
				t.Fatalf("Short write: got %d, want %d", n, len(chunk))
			}
		}

		// Verify Length
		if rb.Len() != len(expected) {
			t.Errorf("Len mismatch: got %d, want %d", rb.Len(), len(expected))
		}

		// Verify Read (this locks the buffer)
		readBack, err := io.ReadAll(rb)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		if !bytes.Equal(readBack, expected) {
			t.Fatalf("Content mismatch.\nCap: %d\nChunk: %d\nExpected: %x\nGot:      %x",
				capacity, chunkSize, expected, readBack)
		}

		// Verify Seek (reset to start)
		offset, err := rb.Seek(0, io.SeekStart)
		if err != nil {
			t.Fatalf("Seek failed: %v", err)
		}
		if offset != 0 {
			t.Errorf("Seek returned offset %d, want 0", offset)
		}

		// Verify Read again
		readBack2, err := io.ReadAll(rb)
		if err != nil {
			t.Fatalf("ReadAll after Seek failed: %v", err)
		}
		if !bytes.Equal(readBack2, expected) {
			t.Fatalf("Content mismatch after seek.\nExpected: %x\nGot:      %x", expected, readBack2)
		}

		// Verify ReadOnly enforcement
		_, err = rb.Write([]byte{0x00})
		if err != ErrReadOnly {
			t.Fatalf("Expected ErrReadOnly after read, got %v", err)
		}
	})
}
