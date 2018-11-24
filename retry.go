package retry

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
)

// DefaultRetryFunc retry when the response status code is in range of 500
// but not 501 which is http.StatusNotImplemented
var DefaultRetryFunc = defaultRetryFunc

// DefaultMaxRetry is default max retry times
const DefaultMaxRetry uint64 = 10

// BackOff is a backoff policy for retrying an operation.
type BackOff = backoff.BackOff

// Func is a function to determine if a retry is needed base on the http.Response
type Func = func(*http.Response) bool

// Log is a flag to enable logging, it's enabled by default
var Log = true

var responseKey = "response"

// Client is a http retry client
type Client struct {
	c *http.Client
}

// New return a new default retry client
func New() *Client {
	return &Client{
		c: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:          500,
				MaxIdleConnsPerHost:   500,
				IdleConnTimeout:       10 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 10 * time.Second,
			},
			Timeout: 30 * time.Second,
		},
	}
}

// Do execute the given request with default backoff policy and default retry func
func (c *Client) Do(r *http.Request) (*http.Response, error) {
	return c.DoWithBackOff(r, DefaultBackOff())
}

// DoWithBackOff execute the given request with the given backoff policy.
// It uses the DefaultRetryFunc which will retry if response status code
// is in range of 500 but not http.StatusNotImplemented.
func (c *Client) DoWithBackOff(r *http.Request, b BackOff) (*http.Response, error) {
	return c.DoWithRetryFunc(r, b, DefaultRetryFunc)
}

// DoWithRetryFunc execute the given request with the given backoff policy.
// A retry is determined by the given retry Func.
func (c *Client) DoWithRetryFunc(r *http.Request, b BackOff, f Func) (*http.Response, error) {
	response := sync.Map{}
	var body []byte
	var err error
	copyBody := false
	if r.Body != nil {
		body, err = ioutil.ReadAll(r.Body)
		if err != nil && err != io.EOF {
			logrus.Errorf("error while reading the request body, given up retrying. Err: %v", err)
			return nil, backoff.Permanent(err)
		}
		r.Body.Close()
		copyBody = true
	}
	op := func() error {
		if copyBody {
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}
		rs, err := c.c.Do(r)
		if err != nil {
			logrus.Errorf("request error, err: %v, need a retry", err)
			return err
		}
		response.Store(responseKey, rs)
		if f(rs) {
			logrus.Errorf("got response from server: %+v, a retry is needed", rs)
			return errors.New("need retry")
		}
		logrus.Infof("executed successfully, response: %+v", rs)
		return nil
	}
	if err := backoff.Retry(op, b); err != nil {
		return nil, fmt.Errorf("failed to retried, err: %v", err)
	}
	v, ok := response.Load(responseKey)
	if !ok {
		return nil, errors.New("executed request successfully, but failed to get response. Propably a bug of retry")
	}
	return v.(*http.Response), nil
}

// DefaultBackOff return a backoff policy with exponential backoff wrapped with a 10-times-max-retry.
func DefaultBackOff() BackOff {
	b := backoff.WithMaxRetries(&backoff.ExponentialBackOff{
		InitialInterval:     1 * time.Second,
		RandomizationFactor: 0,
		Multiplier:          2,
		MaxInterval:         60 * time.Second,
		Clock:               backoff.SystemClock,
	}, DefaultMaxRetry)
	b.Reset()
	return b
}

// defaultRetryFunc retry if the response status code is in range of 500 but not http.StatusNotImplemented.
func defaultRetryFunc(rs *http.Response) bool {
	return rs.StatusCode == 0 || (rs.StatusCode >= 500 && rs.StatusCode != http.StatusNotImplemented)
}
