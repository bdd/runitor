// Copyright (c) Berk D. Demir and the runitor contributors.
// SPDX-License-Identifier: 0BSD
package internal_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "bdd.fi/x/runitor/internal"
)

const (
	TestHandle           = "pingKey/testHandle"
	TestRunId            = "00000000-1111-4000-a000-223344556677"
	TestCreateParamValue = "1"
)

var (
	TestPingParamsNone          = PingParams{}
	TestPingParamsWithRID       = PingParams{RunId: TestRunId}
	TestPingParamsWithCreate    = PingParams{Create: true}
	TestPingParamsWithRIDCreate = PingParams{RunId: TestRunId, Create: true}
	TestPingBody                = []byte("Test Ping Body")
)

// Tests if APIClient makes requests with the expected method, content-type,
// and user-agent.
func TestPostRequest(t *testing.T) {
	t.Parallel()

	const (
		expMethod = "POST"
		expCT     = "text/plain"
		expUA     = "test-user-agent"
	)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expMethod {
			t.Errorf("expected to receive http method %s, got %s", expMethod, r.Method)
		}

		reqCT := r.Header.Get("content-type")
		if reqCT != expCT {
			t.Errorf("expected to receive content-type %s, got %s", expCT, reqCT)
		}

		if r.UserAgent() != expUA {
			t.Errorf("expected client to set header User-Agent: %q, got: %q", expUA, r.UserAgent())
		}
	}))

	defer ts.Close()

	c := &APIClient{
		BaseURL:   ts.URL,
		Client:    ts.Client(),
		UserAgent: expUA,
	}

	_, err := c.PingSuccess(TestHandle, TestPingParamsNone, nil)
	if err != nil {
		t.Fatalf("expected successful Ping, got error: %+v", err)
	}
}

// Tests if APIClient treats HTTP 201 as a successful response.
func TestPost201Response(t *testing.T) {
	t.Parallel()

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	defer ts.Close()

	c := &APIClient{
		BaseURL: ts.URL,
		Client:  ts.Client(),
	}

	_, err := c.PingSuccess(TestHandle, TestPingParamsWithRIDCreate, nil)
	if err != nil {
		t.Fatalf("expected successful Ping, got error: %+v", err)
	}
}

// Tests if request timeout errors and HTTP 5XX responses get retried.
func TestPostRetries(t *testing.T) {
	t.Parallel()

	const SleepToCauseTimeout = 0

	if testing.Short() {
		t.Skip("skipping retry tests with backoff in short mode.")
	}

	backoff := 1 * time.Millisecond
	// clientTimeout needs to give enough time for a slow test runner to complete a TLS handshake.
	// May 2022: less than 25ms might not be enough for some GitHub Actions runs.
	clientTimeout := 200 * time.Millisecond
	sleepTime := clientTimeout

	retryTests := []int{SleepToCauseTimeout}                   // sleep
	retryTests = append(retryTests, RetriableResponseCodes...) // all retriable response codes
	retryTests = append(retryTests, http.StatusOK)             // MUST end with OK

	expectedTries := uint32(len(retryTests))
	var tries uint32

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		try := atomic.AddUint32(&tries, 1)
		if try <= expectedTries {
			status := retryTests[try-1]
			if status == SleepToCauseTimeout {
				time.Sleep(sleepTime)
				return
			}

			w.WriteHeader(status)
		} else {
			t.Fatalf("expected client to try %d times, received %d tries", expectedTries, try)
		}
	}))

	defer ts.Close()

	client := ts.Client()
	client.Timeout = clientTimeout

	c := &APIClient{
		BaseURL: ts.URL,
		Client:  client,
		Retries: uint(expectedTries - 1),
		Backoff: backoff,
	}

	rb := NewRingBuffer(100)
	rb.Write(TestPingBody)
	_, err := c.PingSuccess(TestHandle, TestPingParamsNone, rb)
	if err != nil {
		t.Fatalf("expected successful Ping, got error: %+v", err)
	}

	if tries < expectedTries {
		t.Fatalf("expected client to try %d times, received %d tries", expectedTries, tries)
	}
}

// Tests if APIClient fails right after receiving a nonretriable error.
func TestPostNonRetriable(t *testing.T) {
	t.Parallel()

	status := http.StatusBadRequest
	tries := 0
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if tries++; tries > 1 {
			t.Errorf("expected client to not retry after receiving status code %d", status)
		}
	}))

	defer ts.Close()

	c := &APIClient{
		BaseURL: ts.URL,
		Client:  ts.Client(),
	}

	_, err := c.PingSuccess(TestHandle, TestPingParamsNone, nil)
	if err == nil {
		t.Errorf("expected PingSuccess to return non-nil error after non-retriable API response")
	}
}

// Tests if POST URI is constructed correctly
func TestPostURIConstruction(t *testing.T) {
	t.Parallel()

	type ping func() (*InstanceConfig, error)

	c := &APIClient{}

	testCases := map[string]string{
		"suffix=":     "/",
		"suffix=/":    "/",
		"suffix=//":   "/",
		"suffix=/foo": "/foo",
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testCase := r.Header.Get("test-case")
		expectedPathPrefix, ok := testCases[testCase]
		if !ok {
			t.Fatalf("Unexpected test case %s", testCase)
		}

		expectedPath, _ := url.JoinPath(expectedPathPrefix, TestHandle, "start")
		if r.URL.Path != expectedPath {
			t.Errorf("For test case %s expected to get path %s, got %s\n", testCase, expectedPath, r.URL.Path)
		}
	}))

	defer ts.Close()

	c.Client = ts.Client()
	for testCase := range testCases {
		reqPath := strings.TrimPrefix(testCase, "suffix=")
		c.BaseURL = ts.URL + reqPath
		c.ReqHeaders = map[string]string{"test-case": testCase}
		if _, err := c.PingStart(TestHandle, TestPingParamsNone); err != nil {
			t.Fatalf("Request for test case %s failed: %v", testCase, err)
		}
	}
}

// Tests if Ping{Start,Log,Status} functions hit the correct URI paths.
func TestPostURIs(t *testing.T) {
	t.Parallel()

	type ping func() (*InstanceConfig, error)

	c := &APIClient{}

	testCases := map[string]ping{
		"/start": func() (*InstanceConfig, error) { return c.PingStart(TestHandle, TestPingParamsNone) },
		"":       func() (*InstanceConfig, error) { return c.PingSuccess(TestHandle, TestPingParamsNone, nil) },
		"/fail":  func() (*InstanceConfig, error) { return c.PingFail(TestHandle, TestPingParamsNone, nil) },
		"/log":   func() (*InstanceConfig, error) { return c.PingLog(TestHandle, TestPingParamsNone, nil) },
		"/42":    func() (*InstanceConfig, error) { return c.PingExitCode(TestHandle, TestPingParamsNone, 42, nil) },
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tail := strings.TrimPrefix(r.URL.Path, "/"+TestHandle)
		_, ok := testCases[tail]
		if !ok {
			t.Fatalf("Unexpected request to URL path '%v'", r.URL.Path)
		}

		// TODO(bdd): Find an equivalent replacement for this.
		//            Do we really need it though?
		//
		// body, _ := io.ReadAll(r.Body)
		// r.Body.Close()
		// if string(body) != uriPath {
		// 	t.Errorf("Got a request for '%s' to path '%s'", body, uriPath)
		// }
	}))

	defer ts.Close()

	c.BaseURL = ts.URL
	c.Client = ts.Client()

	for _, fn := range testCases {
		if _, err := fn(); err != nil {
			t.Errorf("Ping failed: %+v", err)
		}
	}
}

// Tests if Ping{Start,Log,Status} functions add the correct query parameters based on PingParams.
func TestPostQueryParams(t *testing.T) {
	t.Parallel()

	c := &APIClient{}

	type TestCase struct {
		Params      PingParams
		QueryString string
	}

	testCases := map[string]TestCase{
		"no-params":      {},
		"rid-only":       {TestPingParamsWithRID, "rid=" + TestRunId},
		"create-only":    {TestPingParamsWithCreate, "create=1"},
		"rid-and-create": {TestPingParamsWithRIDCreate, "create=1&rid=" + TestRunId},
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Header.Get("test-case")
		tc, ok := testCases[name]
		if !ok {
			t.Fatalf("Unexpected test case %s", name)
		}

		if r.URL.RawQuery != tc.QueryString {
			t.Errorf("For test case %s expected to get query string %s, got %s\n", name, tc.QueryString, r.URL.RawQuery)
		}
	}))

	defer ts.Close()

	c.BaseURL = ts.URL
	c.Client = ts.Client()
	for name, tc := range testCases {
		c.ReqHeaders = map[string]string{"test-case": name}
		if _, err := c.PingStart(TestHandle, tc.Params); err != nil {
			t.Fatalf("Request for test case %s failed: %v", name, err)
		}
	}
}

// Tests additional request headers are sent.
func TestPostReqHeaders(t *testing.T) {
	t.Parallel()

	expReqHeaders := map[string]string{
		"foo-header": "foo-val",
		"bar-header": "bar-val",
		"baz-header": "baz-val",
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for expHeader, expVal := range expReqHeaders {
			val := r.Header.Get(expHeader)
			if len(val) == 0 {
				t.Errorf("expected header %s to be set, but wasn't.", expHeader)
			} else if val != expReqHeaders[expHeader] {
				t.Errorf("expected header %s to be set to %s, but got %s.", expHeader, expVal, val)
			}
		}
	}))

	defer ts.Close()

	c := &APIClient{
		BaseURL:    ts.URL,
		Client:     ts.Client(),
		ReqHeaders: expReqHeaders,
	}

	_, err := c.PingSuccess(TestHandle, TestPingParamsNone, nil)
	if err != nil {
		t.Fatalf("expected successful Ping, got error: %+v", err)
	}
}

// Tests if http.DefaultTransport can be type asserted to *http.Transport
// and a TLSClientConfig is set.
func TestNewDefaultTransportWithResumption(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked")
		}
	}()

	tr := NewDefaultTransportWithResumption()
	if tr.TLSClientConfig == nil {
		t.Errorf("TLSClientConfig is not set")
	}
}

// Tests if Content-Length gets set correctly when a RingBuffer is used as
// request body.
func TestContentLengthForRingBufferBody(t *testing.T) {
	t.Parallel()

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Header.Get("Content-Length")
		if v != fmt.Sprintf("%d", len(TestPingBody)) {
			t.Errorf("Content-Length header should be set to %d, but got %s", len(TestPingBody), v)
		}
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("could not read request body")
		}
		if string(reqBody) != string(TestPingBody) {
			t.Errorf("request body does not match expected body")
		}
	}))

	defer ts.Close()

	c := &APIClient{
		BaseURL: ts.URL,
		Client:  ts.Client(),
	}

	rb := NewRingBuffer(100)
	rb.Write(TestPingBody)
	_, err := c.PingSuccess(TestHandle, TestPingParamsNone, rb)
	if err != nil {
		t.Fatalf("ping failed: %+v", err)
	}
}

// Tests if a RingBuffer request body is resubmitted on a 307 redirect.
func TestResubmitRingBufferBody(t *testing.T) {
	t.Parallel()

	pingReceived := false

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("could not read request body")
		}
		if !bytes.Equal(reqBody, TestPingBody) {
			t.Errorf("request body does not match expected body: %s", reqBody)
		}
		if r.URL.Path == "/redirect-target" {
			pingReceived = true
		} else {
			w.Header().Set("Location", "/redirect-target")
			w.WriteHeader(307)
		}
	}))

	defer ts.Close()

	c := &APIClient{
		BaseURL: ts.URL,
		Client:  ts.Client(),
	}

	rb := NewRingBuffer(100)
	rb.Write(TestPingBody)
	_, err := c.PingSuccess(TestHandle, TestPingParamsNone, rb)
	if err != nil {
		t.Fatalf("ping failed: %+v", err)
	}

	if !pingReceived {
		t.Fatalf("ping request succeeded, but redirect target was never called")
	}
}
