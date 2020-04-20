package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

const DefaultAPIBaseURL = "https://hc-ping.com"

var DefaultAPIClient = &APIClient{
	BaseURL:    DefaultAPIBaseURL,
	HTTPClient: http.DefaultClient,
}

type PingType string

const (
	Success PingType = ""
	Failure          = "/fail"
	Start            = "/start"
)

type Check struct {
	UUID      string
	APIClient *APIClient
}

func NewCheck(UUID string) *Check {
	return &Check{
		UUID:      UUID,
		APIClient: DefaultAPIClient,
	}
}

type Pinger interface {
	PingStart(io.Reader) bool
	PingSuccess(io.Reader) bool
	PingFailure(io.Reader) bool
}

func (c *Check) PingStart(body io.Reader) bool {
	return c.ping(Start, body)
}

func (c *Check) PingSuccess(body io.Reader) bool {
	return c.ping(Success, body)
}

func (c *Check) PingFailure(body io.Reader) bool {
	return c.ping(Failure, body)
}

func (c *Check) ping(t PingType, body io.Reader) (ok bool) {
	url := fmt.Sprintf("%s/%s%s", c.APIClient.BaseURL, c.UUID, string(t))
	res, err := c.APIClient.HTTPClient.Post(url, "text/plain", body)
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode == 200 {
		ok = true
	}

	return
}
