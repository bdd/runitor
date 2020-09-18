package internal_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	. "bdd.fi/x/runitor/internal"

	"net/http"
	"net/http/httptest"
	"testing"
)

const TestUUID string = "test-uuid"

// Tests if APIClient makes requests with the expected method, content-type,
// and user-agent.
func TestPostRequest(t *testing.T) {
	const expMethod = "POST"
	const expCT = "text/plain"
	const expUA = "test-user-agent"

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
	if testing.Short() {
		t.Skip("skipping retry tests with backoff in short mode.")
	}

	var retryTests = []int{
		http.StatusRequestTimeout,
		500,
		599,
		200, // must end with 200
	}

	expectedTries := len(retryTests)

	tries := 0
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tries++; tries <= expectedTries {
			w.WriteHeader(retryTests[tries-1])
		} else {
			t.Fatalf("expected client to try %d times, received %d tries", tries-1, tries)
		}
	}))

	defer ts.Close()

	c := &APIClient{
		BaseURL: ts.URL,
		Client:  ts.Client(),
		Retries: expectedTries - 1,
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
	var testCases = map[string]ping{
		fmt.Sprintf("/%s/%s", TestUUID, "start"): c.PingStart,
		fmt.Sprintf("/%s", TestUUID):             c.PingSuccess,
		fmt.Sprintf("/%s/%s", TestUUID, "fail"):  c.PingFailure,
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
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
