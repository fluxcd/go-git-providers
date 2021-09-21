/*
Copyright 2021 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stash

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-cleanhttp"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

const (
	defaultRetries      = 5
	defaultHost         = "localhost"
	headerRateLimit     = "RateLimit-Limit"
	headerRateReset     = "RateLimit-Reset"
	defaultTimeout      = 10 * time.Second
	defaultRetryWaitMin = 100 * time.Millisecond
	defaultRetryWaitMax = 400 * time.Millisecond
)

var (
	// ErrorUnexpectedStatusCode is used when an unexpected status code is returned.
	// The expected status code are
	// - 200 for a successful request
	// - 201 for a successful creation
	// - 202 for a successful request that is in progress
	// - 204 for a successful request that returns no content
	// - 400 for a request that is malformed
	// - 404 for a request that fails due to not found
	ErrorUnexpectedStatusCode = errors.New("unexpected status code")

	defaultTransport = cleanhttp.DefaultPooledTransport()
)

// Doer is the interface that wraps the basic Do method.
//
// Do makes an http request for req.
// It returns the response body as a byte slice and any error encountered.
// It also return a pointer to the response object.
// Do must not modify the request object.
type Doer interface {
	Do(req *http.Request) ([]byte, *http.Response, error)
}

// ClientOptionsFunc are options for the Client.
// It can be used for example to setup a custom http Client.
type ClientOptionsFunc func(Client *Client)

type service struct{ Client *Client }

// A Client is a retryable HTTP Client.
// The Client will automatically retry when it encounters recoverable errors.
// The Client will also retry when it encounters a 429 Too Many Requests status.
// The retry logic can be disabled by setting the DisableRetries option to true.
// This Client is safe to use across multiple goroutines.
// The Client will rate limit the number of requests per second.
type Client struct {
	// Client is retryable http Client.
	Client *retryablehttp.Client
	// DisableRetries is used to disable the default retry logic.
	DisableRetries bool
	// configureLimiterOnce is used to make sure the limiter is configured exactly once.
	configureLimiterOnce sync.Once
	// limiter is used to limit API calls and prevent 429 responses.
	limiter RateLimiter
	// BaseURL is the base URL for API requests.
	BaseURL *url.URL
	//HeaderFields is the header fields for all requests.
	HeaderFields *http.Header
	// Logger is the logger used to log the request and response.
	Logger *logr.Logger

	// Services are used to communicate with the different stash endpoints.
	Users    Users
	Groups   Groups
	Projects Projects
}

// RateLimiter is the interface that wraps the basic Wait method.
// All rate limiters must implement this interface.
type RateLimiter interface {
	Wait(context.Context) error
}

// NewClient returns a new Client given a host name an optional http.Client, a logger, http.Header and ClientOptionsFunc.
// If the http.Client is nil, a default http.Client is used.
// If the http.Header is nil, a default http.Header is used.
// ClientOptionsFunc is an optional function and can be used to configure the client.
// Example:
//  c, err := NewClient(
//  	&http.Client {
//  		Transport: defaultTransport,
//  		Timeout:   defaultTimeout,
//  		}, "https://github.com",
//  		&http.Header {
//  			"Content-Type": []string{"application/json"},
//  		},
//  		&logr.Logger{},
//  		func(c *Client) {
//  			c.DisableRetries = true
//  	})
func NewClient(httpClient *http.Client, host string, header *http.Header, logger *logr.Logger, opts ...ClientOptionsFunc) (*Client, error) {
	if host == "" {
		return nil, errors.New("host is required")
	}

	if logger == nil {
		return nil, errors.New("logger is required")
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Transport: defaultTransport,
			Timeout:   defaultTimeout,
		}
	}

	c := &Client{
		Logger: logger,
	}

	c.Client = &retryablehttp.Client{
		Backoff:      c.retryHTTPBackoff,
		CheckRetry:   c.retryHTTPCheck,
		ErrorHandler: retryablehttp.PassthroughErrorHandler,
		HTTPClient:   httpClient,
		RetryWaitMin: defaultRetryWaitMin,
		RetryWaitMax: defaultRetryWaitMax,
		RetryMax:     defaultRetries,
	}

	for _, opt := range opts {
		opt(c)
	}

	err := c.setBaseURL(host)
	if err != nil {
		return nil, err
	}

	if header == nil {
		header = &http.Header{}
	}

	c.HeaderFields = header

	c.Users = &UsersService{Client: c}
	c.Groups = &GroupsService{Client: c}
	c.Projects = &ProjectsService{Client: c}

	return c, nil
}

func (c *Client) setBaseURL(host string) error {
	h := host
	if !strings.Contains(h, "http") && !strings.Contains(h, "https") {
		h = fmt.Sprintf("https://%s", host)
	}

	url, err := url.ParseRequestURI(h)
	if err != nil {
		return fmt.Errorf("failed to parse host %s to url, %w", h, err)
	}

	c.BaseURL = url

	return nil
}

// retryHTTPCheck provides a callback for Client.CheckRetry which
// will retry both rate limit (429) and server (>= 500) errors as well as other recoverable errors.
func (c *Client) retryHTTPCheck(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "connection refused") ||
			strings.Contains(errMsg, "http2: no cached connection was available") ||
			strings.Contains(errMsg, "net/http: TLS handshake timeout") ||
			strings.Contains(errMsg, "i/o timeout") ||
			strings.Contains(errMsg, "unexpected EOF") ||
			strings.Contains(errMsg, "Client.Timeout exceeded while awaiting headers") {
			return true, nil
		}

		return false, err
	}

	if !c.DisableRetries && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) {
		return true, nil
	}

	return false, nil
}

// retryHTTPBackoff provides a generic callback for Client.Backoff which
// will pass through all calls based on the status code of the response.
func (c *Client) retryHTTPBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	// Use the rate limit backoff function when we are rate limited.
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		return rateLimitBackoff(min, max, attemptNum, resp)
	}

	// Set custom duration when we experience a service interruption.
	min = 700 * time.Millisecond
	max = 900 * time.Millisecond

	return retryablehttp.LinearJitterBackoff(min, max, attemptNum, resp)
}

// rateLimitBackoff provides a callback for Client.Backoff which will use the
// RateLimit-Reset header to determine the time to wait. We add some jitter
// to prevent a thundering herd.
//
// min and max are mainly used for bounding the jitter that will be added to
// the reset time retrieved from the headers. But if the final wait time is
// less then min, min will be used instead.
func rateLimitBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	// rnd is used to generate pseudo-random numbers.
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	// First create some jitter bounded by the min and max durations.
	jitter := time.Duration(rnd.Float64() * float64(max-min))

	if resp != nil {
		if v := resp.Header.Get(headerRateReset); v != "" {
			if reset, _ := strconv.ParseInt(v, 10, 64); reset > 0 {
				// Only update min if the given time to wait is longer.
				if wait := time.Until(time.Unix(reset, 0)); wait > min {
					min = wait
				}
			}
		}
	}

	return min + jitter
}

// configureLimiter configures the rate limiter.
func (c *Client) configureLimiter() error {
	// Set default values for when rate limiting is disabled.
	limit := rate.Inf
	burst := 0

	defer func() {
		// Create a new limiter using the calculated values.
		c.limiter = rate.NewLimiter(limit, burst)
	}()

	// Create a new request.
	req, err := http.NewRequest("GET", c.BaseURL.String(), nil)
	if err != nil {
		return err
	}

	// Make a single request to retrieve the rate limit headers.
	resp, err := c.Client.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if v := resp.Header.Get(headerRateLimit); v != "" {
		if rateLimit, _ := strconv.ParseFloat(v, 64); rateLimit > 0 {
			// The rate limit is based on requests per minute, so for our limiter to
			// work correctly we divide the limit by 60 to get the limit per second.
			rateLimit /= 60
			// Configure the limit and burst using a split of 2/3 for the limit and
			// 1/3 for the burst. This enables clients to burst 1/3 of the allowed
			// calls before the limiter kicks in. The remaining calls will then be
			// spread out evenly using intervals of time.Second / limit which should
			// prevent hitting the rate limit.
			limit = rate.Limit(rateLimit * 0.66)
			burst = int(rateLimit * 0.33)
		}
	}

	return nil
}

// RequestOptions defines the optional parameters for the request.
type RequestOptions struct {
	// Body is the request body.
	Body io.Reader
	// Header is the request header.
	Header http.Header
	// Query is the request query.
	Query url.Values
}

// RequestOptionFunc is a function that set request options.
type RequestOptionFunc func(*RequestOptions)

// WithQuery adds the query parameters to the request.
func WithQuery(query url.Values) RequestOptionFunc {
	return func(r *RequestOptions) {
		if query != nil {
			r.Query = query
		}
	}
}

// WithBody adds the body to the request.
func WithBody(body io.Reader) RequestOptionFunc {
	return func(r *RequestOptions) {
		if body != nil {
			r.Body = body
		}
	}
}

// WithHeader adds the headers to the request.
func WithHeader(header http.Header) RequestOptionFunc {
	return func(r *RequestOptions) {
		if header != nil {
			r.Header = header
		}
	}
}

// NewRequest creates a request, and returns an http.Request and an error,
// given a path and optional query, body, and header. Use the currying functions provided to pass in the request options
// A relative URL path can be provided in path, in which case it is resolved relative to the base URL of the Client.
// Relative URL paths should always be specified without a preceding slash.
// If specified, the value pointed to by body is JSON encoded and included as the request body.
func (c *Client) NewRequest(ctx context.Context, method string, path string, opts ...RequestOptionFunc) (*http.Request, error) {
	u := *c.BaseURL
	unescaped, err := url.PathUnescape(path)
	if err != nil {
		return nil, err
	}

	// Set the encoded path data
	u.RawPath = c.BaseURL.Path + path
	u.Path = c.BaseURL.Path + unescaped

	if method == "" {
		method = http.MethodGet
	}

	r := RequestOptions{}
	for _, opt := range opts {
		opt(&r)
	}

	//var bodyReader io.ReadCloser
	//if (method == http.MethodPost || method == http.MethodPut) && body != nil {
	//	jsonBody, e := json.Marshal(body)
	//	if e != nil {
	//		return nil, fmt.Errorf("failed to marshall request body, %w", e)
	//	}
	//
	//	bodyReader = io.NopCloser(bytes.NewReader(jsonBody))
	//
	//	c.Logger.V(2).Info("request", "body", string(jsonBody))
	//}

	if r.Query == nil {
		r.Query = url.Values{}
	}

	u.RawQuery = r.Query.Encode()

	req, err := http.NewRequest(method, u.String(), r.Body)
	if err != nil {
		return req, fmt.Errorf("failed create request for %s %s, %w", method, u.String(), err)
	}

	req = req.WithContext(ctx)

	if c.HeaderFields != nil {
		for k, v := range *c.HeaderFields {
			for _, s := range v {
				req.Header.Add(k, s)
			}
		}
	}

	if r.Header != nil {
		for k, v := range r.Header {
			for _, s := range v {
				req.Header.Add(k, s)
			}
		}
	}

	return req, nil
}

// Do performs a request, and returns an http.Response and an error given an http.Request.
// For an outgoing Client request, the context controls the entire lifetime of a reques:
// obtaining a connection, sending the request, checking errors and retrying.
// The response body is closed.
func (c *Client) Do(request *http.Request) ([]byte, *http.Response, error) {
	// If not yet configured, try to configure the rate limiter. Fail
	// silently as the limiter will be disabled in case of an error.
	c.configureLimiterOnce.Do(func() { c.configureLimiter() })

	// Wait will block until the limiter can obtain a new token.
	err := c.limiter.Wait(request.Context())
	if err != nil {
		return nil, nil, err
	}

	c.Logger.V(2).Info("request", "method", request.Method, "url", request.URL)

	req, err := retryablehttp.FromRequest(request)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, resp, nil
	}

	resBytes, err := getRespBody(resp)
	if err != nil {
		return nil, resp, err
	}

	if resp.StatusCode == http.StatusOK || (resp.StatusCode == http.StatusCreated && request.Method == http.MethodPost) || (resp.StatusCode == http.StatusNoContent && request.Method == http.MethodDelete) ||
		(resp.StatusCode == http.StatusAccepted && request.Method == http.MethodDelete) || (resp.StatusCode == http.StatusNoContent && request.Method == http.MethodPut) || resp.StatusCode == http.StatusBadRequest {
		return resBytes, resp, nil
	}

	return nil, resp, fmt.Errorf("request %s %s returned status code: %s, %w", request.Method, request.URL, resp.Status, ErrorUnexpectedStatusCode)
}

// getRespBody is used to obtain the response body as a []byte.
func getRespBody(resp *http.Response) ([]byte, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	return data, nil
}
