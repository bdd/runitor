package healthchecks

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bdd/hcpingrun/pkg/api"
)

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

const (
	// Default Healthchecks API address
	DefaultBaseURL = "https://hc-ping.com"
	// Default HTTP client timeout
	DefaultTimeout = 5 * time.Second
	// Default number of tries
	DefaultMaxTries = 3
)

var DefaultAPIClient = &APIClient{
	BaseURL:  DefaultBaseURL,
	MaxTries: DefaultMaxTries,
	Client:   &http.Client{Timeout: DefaultTimeout},
}

type pingType string

// Ping type decides the URL path for ping
const (
	success pingType = ""
	failure          = "/fail"
	start            = "/start"
)

func (c *APIClient) PingStart(UUID string, body io.Reader) error {
	return c.ping(UUID, body, start)
}

func (c *APIClient) PingSuccess(UUID string, body io.Reader) error {
	return c.ping(UUID, body, success)
}

func (c *APIClient) PingFailure(UUID string, body io.Reader) error {
	return c.ping(UUID, body, failure)
}

func (c *APIClient) ping(UUID string, body io.Reader, t pingType) error {
	apiURL := fmt.Sprintf("%s/%s%s", c.BaseURL, UUID, string(t))
	tries := 0
Try:
	// Linear backoff at second granularity
	time.Sleep(time.Duration(tries) * time.Second)
	if tries++; tries >= c.MaxTries {
		return &api.PingError{
			Type: string(t),
			Err:  fmt.Errorf("max tries (%d) reached", c.MaxTries),
		}
	}
	res, err := c.Post(apiURL, "text/plain", body)
	if err != nil {
		// Retry timeout and temporary kind of errors
		if v, ok := err.(*url.Error); ok && (v.Timeout() || v.Temporary()) {
			goto Try
		}
		return err // non-recoverable
	}

	switch {
	case res.StatusCode == 200:
		return nil
	case res.StatusCode == 408 || res.StatusCode >= 500:
		goto Try
	default:
		return &api.PingError{
			Type: string(t),
			Err:  fmt.Errorf("nonretrieable API response: %s", res.Status),
		}
	}
}
