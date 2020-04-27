package api

import (
	"fmt"
	"io"
)

type Pinger interface {
	PingStart(string, io.Reader) error
	PingSuccess(string, io.Reader) error
	PingFailure(string, io.Reader) error
}

type PingError struct {
	Type string // start, success, failure
	Err  error
}

func (e *PingError) Error() string {
	return fmt.Sprintf("%s", e.Err.Error())
}
