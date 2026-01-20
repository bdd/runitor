// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package internal

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrCronFieldCount = errors.New("expected 5 fields")
	ErrCronInvalidStep = errors.New("invalid step")
	ErrCronPositiveStep = errors.New("step must be positive")
	ErrCronRangeStart = errors.New("invalid range start")
	ErrCronRangeEnd = errors.New("invalid range end")
	ErrCronInvalidValue = errors.New("invalid value")
	ErrCronOutOfRange = errors.New("value out of range")
	ErrCronRangeOrder = errors.New("range start > end")
)

// Cron represents a parsed cron schedule.
type Cron struct {
	minutes [60]bool
	hours   [24]bool
	dom     [32]bool // 1-31
	months  [13]bool // 1-12
	dow     [8]bool  // 0-7 (7 is Sunday, aliased to 0)
	domAll  bool
	dowAll  bool
}

// ParseCron parses a standard 5-field cron string.
// Supported features:
// - lists (1,2,3)
// - ranges (1-5)
// - steps (*/5, 1-10/2)
// - * (all)
// - day of week: 0-7 (Sunday=0 or 7)
func ParseCron(s string) (*Cron, error) {
	fields := strings.Fields(s)
	if len(fields) != 5 {
		return nil, fmt.Errorf("%w, got %d", ErrCronFieldCount, len(fields))
	}

	c := &Cron{}
	var err error

	if _, err = parseField(fields[0], 0, 59, c.minutes[:]); err != nil {
		return nil, fmt.Errorf("parsing minutes: %w", err)
	}
	if _, err = parseField(fields[1], 0, 23, c.hours[:]); err != nil {
		return nil, fmt.Errorf("parsing hours: %w", err)
	}
	if c.domAll, err = parseField(fields[2], 1, 31, c.dom[:]); err != nil {
		return nil, fmt.Errorf("parsing dom: %w", err)
	}
	if _, err = parseField(fields[3], 1, 12, c.months[:]); err != nil {
		return nil, fmt.Errorf("parsing months: %w", err)
	}
	// Allow 0-7 for Day of Week
	if c.dowAll, err = parseField(fields[4], 0, 7, c.dow[:]); err != nil {
		return nil, fmt.Errorf("parsing dow: %w", err)
	}

	// Handle 7 as Sunday (alias to 0)
	if c.dow[7] {
		c.dow[0] = true
	}

	return c, nil
}

// parseField parses a cron field and returns true if the field was literally "*".
func parseField(s string, min, max int, dest []bool) (bool, error) {
	// If field is "*", set all to true.
	if s == "*" {
		for i := min; i <= max; i++ {
			dest[i] = true
		}
		return true, nil
	}

	parts := strings.Split(s, ",")
	for _, part := range parts {
		step := 1
		rangeStr := part

		if i := strings.Index(part, "/"); i >= 0 {
			stepStr := part[i+1:]
			var err error
			step, err = strconv.Atoi(stepStr)
			if err != nil {
				return false, fmt.Errorf("%w %q: %w", ErrCronInvalidStep, stepStr, err)
			}
			if step <= 0 {
				return false, ErrCronPositiveStep
			}
			rangeStr = part[:i]
		}

		var start, end int
		var err error

		if rangeStr == "*" {
			start, end = min, max
		} else if i := strings.Index(rangeStr, "-"); i >= 0 {
			startStr := rangeStr[:i]
			endStr := rangeStr[i+1:]
			start, err = strconv.Atoi(startStr)
			if err != nil {
				return false, fmt.Errorf("%w %q: %w", ErrCronRangeStart, startStr, err)
			}
			end, err = strconv.Atoi(endStr)
			if err != nil {
				return false, fmt.Errorf("%w %q: %w", ErrCronRangeEnd, endStr, err)
			}
		} else {
			start, err = strconv.Atoi(rangeStr)
			if err != nil {
				return false, fmt.Errorf("%w %q: %w", ErrCronInvalidValue, rangeStr, err)
			}
			end = start
		}

		if start < min || end > max {
			return false, fmt.Errorf("%w [%d, %d]", ErrCronOutOfRange, min, max)
		}
		if start > end {
			return false, ErrCronRangeOrder
		}

		for i := start; i <= end; i += step {
			dest[i] = true
		}
	}
	return false, nil
}

// Next returns the next scheduled time after t.
// It assumes t is in the desired location (timezone).
func (c *Cron) Next(t time.Time) time.Time {
	// Start checking from the next minute
	next := t.Truncate(time.Minute).Add(time.Minute)

	// To prevent infinite loops (though unlikely with valid cron), limit search to a few years.
	// 5 years seems safe.
	limit := next.AddDate(5, 0, 0)

	for next.Before(limit) {
		// Month check
		month := int(next.Month())
		if !c.months[month] {
			// Move to start of next month
			// Simply adding 1 to month logic handles year rollover
			next = time.Date(next.Year(), next.Month()+1, 1, 0, 0, 0, 0, next.Location())
			continue
		}

		// Day check
		dom := next.Day()
		dow := int(next.Weekday())

		// Logic:
		// If both DOM and DOW are restricted (not *), then match if EITHER matches.
		// If only one is restricted, match that one (the other is *).
		// If both are *, match everything (AND/OR doesn't matter).
		isDomRestricted := !c.domAll
		isDowRestricted := !c.dowAll

		matchDom := c.dom[dom]
		matchDow := c.dow[dow]

		matchDay := false
		if isDomRestricted && isDowRestricted {
			matchDay = matchDom || matchDow
		} else {
			matchDay = matchDom && matchDow
		}

		if !matchDay {
			// Advance day
			next = time.Date(next.Year(), next.Month(), next.Day()+1, 0, 0, 0, 0, next.Location())
			continue
		}

		// Hour check
		hour := next.Hour()
		if !c.hours[hour] {
			next = next.Add(time.Hour)
			// Reset minute
			next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), 0, 0, 0, next.Location())
			continue
		}

		// Minute check
		minute := next.Minute()
		if !c.minutes[minute] {
			next = next.Add(time.Minute)
			continue
		}

		return next
	}
	return time.Time{} // Should not happen
}

