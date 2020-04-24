package api

import "io"

type Pinger interface {
	PingStart(io.Reader) bool
	PingSuccess(io.Reader) bool
	PingFailure(io.Reader) bool
}
