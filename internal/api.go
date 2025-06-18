// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
//
// Package internal contains healthchecks.io HTTP API client implementation for
// cmd/runitor's use.
//
// It is not intended to be used as a standalone client package.
package internal

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	urlpkg "net/url"
	"strconv"
	"time"
)

const (
	// Default Healthchecks API address.
	DefaultBaseURL = "https://hc-ping.com"
	// Default HTTP client timeout.
	DefaultTimeout = 5 * time.Second
	// Default number of retries.
	DefaultRetries = 2
	// Header to relay instance's ping body limit.
	PingBodyLimitHeader = "Ping-Body-Limit"
)

var (
	ErrNonRetriable = errors.New("nonretriable error response")
	ErrMaxTries     = errors.New("max tries reached")
	// HTTP response codes eligible for retries.
	RetriableResponseCodes = []int{
		http.StatusRequestTimeout,      // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
	}
)

// NewDefaultTransportWithResumption returns an http.Transport based on
// http.DefaultTransport with a TLS Client Session Cache, to enable TLS session
// resumption.
func NewDefaultTransportWithResumption() *http.Transport {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		panic("cannot assert DefaultTranport to *Transport")
	}

	t.TLSClientConfig = &tls.Config{
		ClientSessionCache: tls.NewLRUClientSessionCache(1),
	}

	return t
}

func retriableResponse(code int) bool {
	for _, i := range RetriableResponseCodes {
		if code == i {
			return true
		}
	}

	return false
}

// Pinger is the interface to Healthchecks.io pinging API
// https://healthchecks.io/docs/http_api/
type Pinger interface {
	PingStart(handle string, params PingParams) (*InstanceConfig, error)
	PingLog(handle string, params PingParams, body io.ReadSeeker) (*InstanceConfig, error)
	PingSuccess(handle string, params PingParams, body io.ReadSeeker) (*InstanceConfig, error)
	PingFail(handle string, params PingParams, body io.ReadSeeker) (*InstanceConfig, error)
	PingExitCode(handle string, params PingParams, exitCode int, body io.ReadSeeker) (*InstanceConfig, error)
}

type PingParams struct {
	RunId  string
	Create bool
}

// APIClient holds API endpoint URL, client behavior configuration, and embeds http.Client.
type APIClient struct {
	// BaseURL is the base URL of Healthchecks API instance
	BaseURL string // BaseURL of the Healthchecks API

	// Retries is the number of times the pinger will retry an API request
	// if it fails with a timeout or temporary kind of error, or an HTTP
	// status of 408, 429, 500, ... (see RetriableResponseCodes)
	Retries uint

	// UserAgent, when non-empty, is the value of 'User-Agent' HTTP header
	// for outgoing requests.
	UserAgent string

	// Backoff is the duration used as the unit of linear backoff.
	Backoff time.Duration

	// ReqHeaders is a map of additional headers to be sent with every request.
	ReqHeaders map[string]string

	// Embed
	*http.Client
}

// InstanceConfig holds the instance specific configuration parameters received
// as HTTP headers to ping requests.
type InstanceConfig struct {
	PingBodyLimit Optional[uint]
}

// FromResponse populates InstanceConfig values from a ping response.
func (c *InstanceConfig) FromResponse(resp *http.Response) {
	strval := resp.Header.Get(PingBodyLimitHeader)
	if len(strval) == 0 {
		return
	}

	// uint32 should be enough for everyone(tm)
	val, err := strconv.ParseUint(strval, 10, 32)
	if err != nil {
		return
	}

	c.PingBodyLimit = Some(uint(val))
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
func (c *APIClient) Post(url, contentType string, body io.ReadSeeker) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	// net.http.NewRequestWithContext automatically sets the content-length
	// header for *bytes.Buffer, *bytes.Reader, and *strings.Reader body
	// types.
	// We set it explicitly for RingBuffer bodies and avoid chunked
	// transfer encoding.
	if rb, ok := body.(*RingBuffer); ok {
		req.ContentLength = int64(rb.Len())
		if req.ContentLength == 0 {
			req.Body = nil
		} else {
			req.GetBody = func() (io.ReadCloser, error) {
				rb.Seek(0, io.SeekStart)
				return io.NopCloser(rb), nil
			}
		}
	}

	if c.ReqHeaders != nil {
		for k, v := range c.ReqHeaders {
			req.Header.Set(k, v)
		}
	}

	req.Header.Set("Content-Type", contentType)

	if len(c.UserAgent) > 0 {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	backoffStep := c.Backoff
	if backoffStep == 0 {
		backoffStep = time.Second
	}

	var tries uint
	goto Try
Retry:
	if req.GetBody != nil {
		req.Body, err = req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("req.GetBody: %v", err)
		}
	}
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
			goto Retry
		}

		// non-recoverable
		return
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		return
	case resp.StatusCode == http.StatusCreated:
		return
	case retriableResponse(resp.StatusCode):
		code := resp.StatusCode
		text := http.StatusText(code)
		err = fmt.Errorf("%d %s", code, text)
		goto Retry
	default:
		err = fmt.Errorf("%w: %s", ErrNonRetriable, resp.Status)
		return
	}
}

// PingStart sends a start ping for the check handle.
func (c *APIClient) PingStart(handle string, params PingParams) (*InstanceConfig, error) {
	return c.ping(handle, params, "start", nil)
}

// PingSuccess sends a success ping for the check handle and attaches body as
// the logged context.
func (c *APIClient) PingSuccess(handle string, params PingParams, body io.ReadSeeker) (*InstanceConfig, error) {
	return c.ping(handle, params, "", body)
}

// PingFail sends a failure ping for the check handle and attaches body as the
// logged context.
func (c *APIClient) PingFail(handle string, params PingParams, body io.ReadSeeker) (*InstanceConfig, error) {
	return c.ping(handle, params, "fail", body)
}

// PingLog sends a logging only ping for the check handle and attaches body as
// the logged context.
func (c *APIClient) PingLog(handle string, params PingParams, body io.ReadSeeker) (*InstanceConfig, error) {
	return c.ping(handle, params, "log", body)
}

// PingExitCode sends the exit code of the monitored command for the check handle
// and attaches body as the logged context.
func (c *APIClient) PingExitCode(handle string, params PingParams, exitCode int, body io.ReadSeeker) (*InstanceConfig, error) {
	return c.ping(handle, params, fmt.Sprintf("%d", exitCode), body)
}

func (c *APIClient) ping(handle string, params PingParams, typePath string, body io.ReadSeeker) (*InstanceConfig, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}

	u.Path, err = url.JoinPath(u.Path, handle, typePath)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	if len(params.RunId) > 0 {
		q.Add("rid", params.RunId)
	}
	if params.Create == true {
		q.Add("create", "1")
	}

	if len(q) > 0 {
		u.RawQuery = q.Encode()
	}

	resp, err := c.Post(u.String(), "text/plain", body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	icfg := &InstanceConfig{}
	icfg.FromResponse(resp)

	return icfg, nil
}
