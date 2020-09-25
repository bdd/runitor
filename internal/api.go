// Package internal contains healthchecks.io HTTP API client implementation for
// cmd/runitor's use.
//
// It is not intended to be used as a standalone client package.
package internal

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"time"
)

const (
	// Default Healthchecks API address.
	DefaultBaseURL = "https://hc-ping.com"
	// Default HTTP client timeout.
	DefaultTimeout = 5 * time.Second
	// Default number of retries.
	DefaultRetries = 2
)

var (
	ErrNonRetriable = errors.New("nonretriable error response")
	ErrMaxTries     = errors.New("max tries reached")
)

// Pinger is the interface to Healthchecks.io pinging API
// https://healthchecks.io/docs/http_api/
type Pinger interface {
	PingStart(uuid string, body io.Reader) error
	PingSuccess(uuid string, body io.Reader) error
	PingFailure(uuid string, body io.Reader) error
}

// APIClient holds API endpoint URL, client behavior configuration, and embeds http.Client.
type APIClient struct {
	// BaseURL is the base URL of Healthchecks API instance
	BaseURL string // BaseURL of the Healthchecks API

	// Retries is the number of times the pinger will retry an API request
	// if it fails with a timeout or temporary kind of error, or an HTTP
	// status of 408 or 5XX.
	Retries int

	// UserAgent, when non-empty, is the value of 'User-Agent' HTTP header
	// for outgoing requests.
	UserAgent string

	// Backoff is the duration used as the unit of linear backoff.
	Backoff time.Duration

	// Embed
	*http.Client
}

// Post wraps embedded http.Client's Post to implement simple retry logic and
// custom User-Agent header injection.
//
// Retries:
// The implementation is inspired from Curl's. Request timeouts and temporary
// network level errors will be retried. Responses with status codes 408 and
// 5XX are also retried. Unlike Curl's, the backoff implementation is linear
// instead of exponential. First retry waits for 1 second, second one waits for
// 2 seconds, and so on.
//
// User-Agent:
// If c.UserAgent is not empty, it overrides http.Client's default header.
func (c *APIClient) Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)

	if len(c.UserAgent) > 0 {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	backoffStep := c.Backoff
	if backoffStep == 0 {
		backoffStep = time.Second
	}

	tries := 0
Try:
	// Linear backoff at second granularity
	time.Sleep(time.Duration(tries) * backoffStep)

	if tries++; tries > 1+c.Retries {
		err = fmt.Errorf("%w after try %d. last error: %v", ErrMaxTries, tries-1, err)
		return
	}

	resp, err = c.Do(req)
	if err != nil {
		// Retry timeout and temporary kind of errors
		var uerr *urlpkg.Error
		if errors.As(err, &uerr) && (uerr.Timeout() || uerr.Temporary()) {
			goto Try
		}

		// non-recoverable
		return
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		return
	case resp.StatusCode == http.StatusRequestTimeout || (resp.StatusCode >= 500 && resp.StatusCode <= 599):
		goto Try
	default:
		err = fmt.Errorf("%w: %s", ErrNonRetriable, resp.Status)
		return
	}
}

// PingStart sends a Start Ping for the check with passed uuid and attaches
// body as the logged context.
func (c *APIClient) PingStart(uuid string, body io.Reader) error {
	return c.ping(uuid, body, "/start")
}

// PingSuccess sends a Success Ping for the check with passed uuid and attaches
// body as the logged context.
func (c *APIClient) PingSuccess(uuid string, body io.Reader) error {
	return c.ping(uuid, body, "")
}

// PingFailure sends a Fail Ping for the check with passed uuid and attaches
// body as the logged context.
func (c *APIClient) PingFailure(uuid string, body io.Reader) error {
	return c.ping(uuid, body, "/fail")
}

func (c *APIClient) ping(uuid string, body io.Reader, typePath string) error {
	u := fmt.Sprintf("%s/%s%s", c.BaseURL, uuid, typePath)

	resp, err := c.Post(u, "text/plain", body)
	if err != nil {
		return err
	}

	resp.Body.Close()

	return nil
}
