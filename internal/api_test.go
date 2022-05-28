package internal_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	. "bdd.fi/x/runitor/internal"
)

const TestHandle string = "pingKey/testHandle"

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

	_, err := c.PingStatus(TestHandle, 0, nil)
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

	backoff := 10 * time.Millisecond
	// clientTimeout needs to give enough time for a slow test runner to complete a TLS handshake.
	// May 2022: less than 25ms might not be enough for some GitHub Actions runs.
	clientTimeout := 10 * backoff
	sleepTime := clientTimeout + backoff

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

	_, err := c.PingStatus(TestHandle, 0, nil)
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

	_, err := c.PingStatus(TestHandle, 0, nil)
	if err == nil {
		t.Errorf("expected PingStatus to return non-nil error after non-retriable API response")
	}
}

// Tests if Ping{Start,Status} functions hit the correct URI paths.
func TestPostURIs(t *testing.T) {
	t.Parallel()

	type ping func() (*InstanceConfig, error)

	c := &APIClient{}

	uriPrefix := "/" + TestHandle + "/"
	// uriPath -> pingFunction
	testCases := map[string]ping{
		uriPrefix + "start": func() (*InstanceConfig, error) { return c.PingStart(TestHandle) },
		uriPrefix + "0":     func() (*InstanceConfig, error) { return c.PingStatus(TestHandle, 0, nil) },
		uriPrefix + "1":     func() (*InstanceConfig, error) { return c.PingStatus(TestHandle, 1, nil) },
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uriPath := r.URL.Path
		_, ok := testCases[uriPath]
		if !ok {
			t.Fatalf("Unknown URI path '%v' received", uriPath)
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

	_, err := c.PingStatus(TestHandle, 0, nil)
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
