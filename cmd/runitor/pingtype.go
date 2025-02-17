// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package main

import (
	"flag"
	"fmt"
	"strings"
)

// PingType is an enumerator type for PingType* constants.
type PingType int

//go:generate go tool github.com/dmarkham/enumer -type PingType -trimprefix=PingType -transform=kebab
const (
	PingTypeExitCode PingType = iota
	PingTypeSuccess
	PingTypeFail
	PingTypeLog
)

func pingTypeFlag(name string, dflt PingType, usage string) *PingType {
	p := new(PingType)
	*p = dflt

	opts := fmt.Sprintf("%s (default %s)", pingTypeOpts("|"), dflt)
	usage = usage + " (" + opts + ")"
	flag.Func(name, usage, func(s string) (err error) {
		*p, err = PingTypeString(s)
		if err != nil {
			err = fmt.Errorf("recognized options: %s", opts)
		}
		return err
	})

	return p
}

func pingTypeOpts(sep string) string {
	var b strings.Builder
	for _, v := range PingTypeValues() {
		if b.Len() > 0 {
			b.WriteString(sep)
		}
		b.WriteString(v.String())
	}

	return b.String()
}
