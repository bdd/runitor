package internal_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "bdd.fi/x/runitor/internal"
)

const TestUUID string = "test-uuid"

// Tests if APIClient makes requests with the expected method, content-type,
// and user-agent.
func TestPostRequest(t *testing.T) {
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

	err := c.PingSuccess(TestUUID, nil)
	if err != nil {
		t.Fatalf("expected successful Ping, got error: %+v", err)
	}
}

// Tests if request timeout errors and HTTP 5XX responses get retried.
func TestPostRetries(t *testing.T) {
	const SleepToCauseTimeout = 0

	if testing.Short() {
		t.Skip("skipping retry tests with backoff in short mode.")
	}

	backoff := 5 * time.Millisecond
	clientTimeout := backoff

	retryTests := []int{
		SleepToCauseTimeout,
		http.StatusRequestTimeout,
		500,
		599,
		SleepToCauseTimeout,
		200, // must end with 200
	}

	expectedTries := len(retryTests)

	tries := 0
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tries++; tries <= expectedTries {
			status := retryTests[tries-1]
			if status == SleepToCauseTimeout {
				time.Sleep(2 * clientTimeout)
				return
			}

			w.WriteHeader(status)
		} else {
			t.Fatalf("expected client to try %d times, received %d tries", expectedTries, tries-1)
		}
	}))

	defer ts.Close()

	client := ts.Client()
	client.Timeout = backoff

	c := &APIClient{
		BaseURL: ts.URL,
		Client:  client,
		Retries: expectedTries - 1,
		Backoff: backoff,
	}

	err := c.PingSuccess(TestUUID, nil)
	if err != nil {
		t.Fatalf("expected successful Ping, got error: %+v", err)
	}

	if tries < expectedTries {
		t.Fatalf("expected client to try %d times, received %d tries", expectedTries, tries)
	}
}

// Tests if APIClient fails right after receiving a nonretriable error.
func TestPostNonRetriable(t *testing.T) {
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

	err := c.PingSuccess(TestUUID, nil)
	if err == nil {
		t.Errorf("expected PingSuccess to return non-nil error after non-retriable API response")
	}
}

// Tests if Ping{Start,Success,Failure} functions hit the correct URI paths.
func TestPostURIs(t *testing.T) {
	type ping func(string, io.Reader) error

	c := &APIClient{}

	// uriPath -> pingFunction
	testCases := map[string]ping{
		fmt.Sprintf("/%s/%s", TestUUID, "start"): c.PingStart,
		fmt.Sprintf("/%s", TestUUID):             c.PingSuccess,
		fmt.Sprintf("/%s/%s", TestUUID, "fail"):  c.PingFailure,
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()

		uriPath := r.URL.Path
		_, ok := testCases[uriPath]
		if !ok {
			t.Fatalf("Unknown URI path '%v' received", uriPath)
		}

		if string(body) != uriPath {
			t.Errorf("Got a request for '%s' to path '%s'", body, uriPath)
		}
	}))

	defer ts.Close()

	c.BaseURL = ts.URL
	c.Client = ts.Client()

	for path, fn := range testCases {
		if err := fn(TestUUID, strings.NewReader(path)); err != nil {
			t.Errorf("Ping failed: %+v", err)
		}
	}
}

// Tests if http.DefaultTransport can be type asserted to *http.Transport
// and a TLSClientConfig is set.
func TestNewDefaultTransportWithResumption(t *testing.T) {
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
