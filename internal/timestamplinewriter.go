// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package internal

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// TimestampLineWriter prefixes each line with a timestamp.
type TimestampLineWriter struct {
	Now     func() time.Time
	Format  string
	w       io.Writer
	atStart bool
	mu      sync.Mutex
}

// NewTimestampLineWriter wraps w and prefixes each line with a timestamp.
func NewTimestampLineWriter(w io.Writer) *TimestampLineWriter {
	return &TimestampLineWriter{
		Now:     time.Now,
		Format:  time.TimeOnly,
		w:       w,
		atStart: true,
	}
}

func (w *TimestampLineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var written int
	prefix := w.Now().Format(w.Format) + " "

	for line := range bytes.SplitAfterSeq(p, []byte("\n")) {
		if w.atStart && len(line) != 0 {
			if _, err := io.WriteString(w.w, prefix); err != nil {
				return written, err
			}
			w.atStart = false
		}

		n, err := w.w.Write(line)
		written += n
		if err != nil {
			return written, err
		}

		if !w.atStart {
			w.atStart = bytes.HasSuffix(line, []byte("\n"))
		}
	}

	return written, nil
}
