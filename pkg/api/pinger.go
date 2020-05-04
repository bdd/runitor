package api // import "bdd.fi/x/runitor/pkg/api"

import (
	"io"
)

type Pinger interface {
	PingStart(string, io.Reader) error
	PingSuccess(string, io.Reader) error
	PingFailure(string, io.Reader) error
}
