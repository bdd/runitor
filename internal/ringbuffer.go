// Copyright 2020 - 2022, Berk D. Demir and the runitor contributors
// SPDX-License-Identifier: 0BSD
package internal

import (
	"errors"
	"io"
)

// ErrReadOnly is the error returned by Write to indicate the ring
// buffer is in read only mode and will not accept further writes.
var ErrReadOnly = errors.New("read only")

// RingBuffer implements io.ReadWriter interface to a []byte backed ring
// buffer (aka circular buffer).
//
// Can be written to repeatedly until read from.
// At first read, ring buffer becomes read only, refusing further writes with
// ErrReadOnly error.
type RingBuffer struct {
	buf      []byte
	idx      int
	unread   int
	readonly bool
}

// Len returns the length of the ring buffer.
func (r *RingBuffer) Len() int {
	return len(r.buf)
}

// Cap returns the capacity of the ring buffer.
func (r *RingBuffer) Cap() int {
	return cap(r.buf)
}

// Wrapped returns true if the ring buffer overwrote at least one byte.
func (r *RingBuffer) Wrapped() bool {
	return r.Len() == r.Cap() && r.idx > 0
}

func (r *RingBuffer) Write(p []byte) (n int, err error) {
	if r.readonly {
		return 0, ErrReadOnly
	}

	return r.write(p), nil
}

func (r *RingBuffer) write(p []byte) (n int) {
	// grow slice by write size, up to capacity.
	if r.Len() != r.Cap() {
		newlen := r.idx + len(p)
		if newlen > r.Cap() {
			newlen = r.Cap()
		}

		r.buf = r.buf[:newlen]
	}

	// If source is larger than the capacity of the ring buffer, we'll
	// need to overwrite unobservable data. Optimize this by only writing
	// last `r.Cap()` bytes from source.
	if len(p) > r.Cap() {
		// jump over what would be overwritten and count as written
		n = len(p) - r.Cap()
		r.idx = (r.idx + n) % r.Cap()
	}

	for n < len(p) {
		cn := copy(r.buf[r.idx:], p[n:])
		n += cn
		r.idx = (r.idx + cn) % r.Cap()
	}

	return
}

func (r *RingBuffer) Read(p []byte) (n int, err error) {
	if !r.readonly {
		r.readonly = true
		r.unread = r.Len()

		if !r.Wrapped() {
			r.idx = 0
		}
	}

	if r.unread == 0 {
		if len(p) == 0 {
			return 0, nil
		}

		return 0, io.EOF
	}

	return r.read(p), nil
}

func (r *RingBuffer) read(p []byte) (n int) {
	goal := len(p)
	if goal > r.unread {
		goal = r.unread
	}

	for n < goal {
		from := r.idx
		to := from + r.unread
		if to > r.Len() {
			to = r.Len()
		}
		cn := copy(p[n:], r.buf[from:to])
		n += cn
		r.unread -= cn
		r.idx = (r.idx + cn) % r.Cap()
	}

	return
}

// Snapshot returns a clone of r.buf
//
// Currently only intended for testing.
func (r *RingBuffer) Snapshot() []byte {
	c := make([]byte, r.Len())
	copy(c, r.buf)

	return c
}

// NewRingBuffer allocates a new RingBuffer and the backing byte array with
// specified capacity.
func NewRingBuffer(cap int) *RingBuffer {
	return &RingBuffer{buf: make([]byte, 0, cap)}
}
