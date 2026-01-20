// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package internal_test

import (
	"testing"
	"time"

	. "bdd.fi/x/runitor/internal"
)

func dt(Y int, M time.Month, D, h, m int) time.Time {
	return time.Date(Y, M, D, h, m, 0, 0, time.UTC)
}

func verify(t *testing.T, name, expr string, expected ...time.Time) {
	t.Helper()

	start := dt(2026, 1, 1, 0, 0)
	c, err := ParseCron(expr)
	if err != nil {
		t.Fatalf("%s: ParseCron failed: %v", name, err)
	}

	current := start
	for i, want := range expected {
		got := c.Next(current)
		if !got.Equal(want) {
			t.Errorf("%s[%d]: Next(%v) = %v, want %v", name, i, current, got, want)
			return // Stop verifying this sequence on failure
		}
		current = got
	}
}

func TestParseCron(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expr  string
		valid bool
	}{
		{"invalid", false},
		{"* * * * *", true},
		{"* * * *", false},
		{"*/5 * * * *", true},
		{"0 0 1 1 *", true},
		{"00 01 * * *", true},
		{"00 23 01 01 *", true},

		// Although Feb 31 is not a thing, this is technically a
		// correct expression.
		{"00 00 31 02 *", true},

		{"60 * * * *", false},
		{"59 24 * * *", false},
		{"59 23 32 * *", false},
		{"59 23 31 00 *", false},
		{"59 23 31 13 *", false},
		{"59 23 31 12 8", false},
	}

	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			t.Parallel()

			_, err := ParseCron(tc.expr)
			if (err != nil) == tc.valid {
				t.Errorf("ParseCron(%q) error = %v, wantErr %v", tc.expr, err, tc.valid)
			}
		})
	}
}

func TestFirstNext(t *testing.T) {
	t.Parallel()

	verify(t, "Every minute", "* * * * *", dt(2026, 1, 1, 0, 1))
	verify(t, "Every 5 minutes", "*/5 * * * *", dt(2026, 1, 1, 0, 5))
	verify(t, "Specific time", "30 14 * * *", dt(2026, 1, 1, 14, 30))
	verify(t, "Next day", "0 0 * * *", dt(2026, 1, 2, 0, 0))
	verify(t, "Specific DoM", "0 0 5 * *", dt(2026, 1, 5, 0, 0))

	// Jan 1 2026 is a Thu. 2=Fri, 3=Sat, 4=Sun, 5=Mon
	verify(t, "Specific DoW", "0 0 * * 1", dt(2026, 1, 5, 0, 0))

	// Jan 1 is a Thu. First Monday is the 5th.
	// Given time is 00:00, our start time, .Next() cannot match.
	// DoM match fails.
	// On the 5th, DoW match succeeds.
	verify(t, "DoM and DoW union", "0 0 1 * 1", dt(2026, 1, 5, 0, 0))

	// First Sunday is Jan 4 2026
	verify(t, "Sunday as 0", "0 0 * * 0", dt(2026, 1, 4, 0, 0))
	verify(t, "Sunday as 7", "0 0 * * 7", dt(2026, 1, 4, 0, 0))
}

func TestSeqNexts(t *testing.T) {
	t.Parallel()

	// Helper to verify a sequence of Next calls

	verify(t,
		"List of hours",
		"0 2,8,14,20 * * *",
		dt(2026, 1, 1, 2, 0),
		dt(2026, 1, 1, 8, 0),
		dt(2026, 1, 1, 14, 0),
		dt(2026, 1, 1, 20, 0),
		dt(2026, 1, 2, 2, 0),
	)

	verify(t,
		"Range of days",
		"1 0 1-3 * *",
		dt(2026, 1, 1, 0, 1),
		dt(2026, 1, 2, 0, 1),
		dt(2026, 1, 3, 0, 1),
		dt(2026, 2, 1, 0, 1),
	)
}

func TestDoMDoWUnion(t *testing.T) {
	t.Parallel()

	// Test the DoM/DoW union rule specifically.
	// 2026-01-01 is Thu.
	// expression: "0 0 1 * 4" -> DoM=1 OR DoW=4 (Thu)
	// Should match Jan 1 (DoM=1 & DoW=4 - both match).

	start := dt(2025, 12, 31, 23, 59)
	c, _ := ParseCron("0 0 1 * 4")

	// Next should be Jan 1 2026 00:00
	want := dt(2026, 1, 1, 0, 0)
	got := c.Next(start)
	if !got.Equal(want) {
		t.Errorf("1: Next() = %v, want %v", got, want)
	}

	// Next after Jan 1 00:00.
	// Next Thursday is Jan 8.
	// Is there a DoM=1? Feb 1 2026 is Sunday.
	// So next is Jan 8 (Thursday).
	got = c.Next(want)
	want = dt(2026, 1, 8, 0, 0)
	if !got.Equal(want) {
		t.Errorf("2: Next() = %v, want %v", got, want)
	}
}

func TestDoMDoWIntersection(t *testing.T) {
	t.Parallel()

	// If only DoM is restricted: "0 0 1 * *" -> DoM=1 AND DoW=*.
	// Should only match Jan 1.
	start := dt(2026, 1, 1, 0, 0)
	c, _ := ParseCron("0 0 1 * *")

	// Next after Jan 1 should be Feb 1
	got := c.Next(start)
	want := dt(2026, 2, 1, 0, 0)
	if !got.Equal(want) {
		t.Errorf("Next() = %v, want %v", got, want)
	}
}
