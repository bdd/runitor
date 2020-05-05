package internal

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Pinger interface {
	PingStart(string, io.Reader) error
	PingSuccess(string, io.Reader) error
	PingFailure(string, io.Reader) error
}

// APIClient holds API endpoint address and client behavior configuration
// implemented outside of net/http package. It embeds http.Client.
type APIClient struct {
	// BaseURL is the base URL of Healthchecks API instance
	BaseURL string // BaseURL of the Healthchecks API

	// MaxTries is the number of times the pinger will retry an API request
	// if it fails with a timeout or temporary kind of error, or an HTTP
	// status of 408 or 5XX.
	MaxTries int

	// Embed
	*http.Client
}

// Post wraps embedded http.Client's Post to implement simple retry logic.
//
// The implementation is inspired from Curl's. Request timeouts and temporary
// network level errors will be retried. Responses with status codes 408 and
// 5XX are also retried. Unlike Curl's, the backoff implementation is linear
// instead of exponential. First retry waits for 1 second, second one waits for
// 2 seconds, and so on.
func (c *APIClient) Post(_url, contentType string, body io.Reader) (resp *http.Response, err error) {
	tries := 0
Try:
	// Linear backoff at second granularity
	time.Sleep(time.Duration(tries) * time.Second)

	if tries++; tries >= c.MaxTries {
		err = fmt.Errorf("max tries (%d) reached after error: %w", c.MaxTries, err)
		return
	}

	resp, err = c.Client.Post(_url, contentType, body)
	if err != nil {
		// Retry timeout and temporary kind of errors
		if v, ok := err.(*url.Error); ok && (v.Timeout() || v.Temporary()) {
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
		err = fmt.Errorf("nonretrieable API response: %s", resp.Status)
		return
	}
}

const (
	// Default Healthchecks API address
	DefaultBaseURL = "https://hc-ping.com"
	// Default HTTP client timeout
	DefaultTimeout = 5 * time.Second
	// Default number of tries
	DefaultMaxTries = 3
)

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
