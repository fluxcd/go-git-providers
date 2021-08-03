package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-logr/logr"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const (
	maxIdleConns         = 100
	defaultTimeoutSecs   = 10
	defaultRetries       = 30
	defaultRetryInterval = 1
	backoffResetInterval = 10
	httpClientDebug      = "HTTP_CLIENT_DEBUG"
)

var (
	ErrorUnexpectedStatusCode = errors.New("unexpected status code")

	defaultTransport *http.Transport = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConns,
	} // nolint:gochecknoglobals // ok

	DefaultTimeout time.Duration = time.Second * defaultTimeoutSecs // nolint:gochecknoglobals // ok
	Post                         = "POST"                           // nolint:gochecknoglobals // ok
	Put                          = "PUT"                            // nolint:gochecknoglobals // ok
	Delete                       = "DELETE"                         // nolint:gochecknoglobals // ok
	Get                          = "GET"                            // nolint:gochecknoglobals // ok
	DefaultDomain  string        = "localhost"                      // nolint:gochecknoglobals // ok
)

// A Requester makes HTTP requests
//
// Do returns an http.Response and an error.
//
// Except for reading the body, handlers should not modify the
// provided Request parameters.
type Requester interface {
	Do(ctx context.Context, path string, query *url.Values, body interface{}, method *string, header *http.Header) (*http.Response, error)
}

// reqResp hold information relating to an HTTPS request and response.
type reqResp struct {
	Requester
	client       *http.Client
	transport    *http.Transport
	url          *url.URL
	timeout      *time.Duration
	headerFields *http.Header
	log          logr.Logger
}

func (r *reqResp) String() string {
	return fmt.Sprintf("domain:%s}", r.url.String())
}

func NewRequester(domain string, header *http.Header, timeout *time.Duration, client *http.Client, transport http.RoundTripper, log logr.Logger) (Requester, error) {

	if transport == nil {
		transport = defaultTransport
	}

	if client == nil {
		client = &http.Client{Transport: transport}
	}

	if timeout == nil {
		timeout = &DefaultTimeout
	}
	client.Timeout = *timeout

	if len(domain) == 0 {
		return nil, errors.New("domain is required")
	}

	d := domain
	if !strings.Contains(d, "http") && !strings.Contains(d, "https") {
		d = fmt.Sprintf("https://%s", domain)
	}

	url, err := url.ParseRequestURI(d)
	if err != nil {
		return nil, fmt.Errorf("failed to parse domain to url, %w", err)
	}

	if header == nil {
		header = &http.Header{}
	}

	r := reqResp{
		transport:    defaultTransport,
		client:       client,
		headerFields: header,
		timeout:      timeout,
		url:          url,
		log:          log,
	}

	return &r, nil
}

// CloseBody closes the response body.
func CloseBody(resp *http.Response) error {
	if resp != nil {
		if resp.Body != nil {
			e := resp.Body.Close()
			if e != nil {
				return fmt.Errorf("failed to close response body, %w", e)
			}
		}
	}
	return nil
}

// Do creates an HTTP client and sends a request.
// The response is returned
func (r *reqResp) Do(ctx context.Context, path string, query *url.Values, body interface{}, method *string, header *http.Header) (*http.Response, error) { // nolint:funlen,gocognit,gocyclo // ok
	var err error
	var resp *http.Response

	var inputJSON io.ReadCloser

	if method == nil {
		method = &Get
	}
	if (*method == Post || *method == Put) && body != nil {
		jsonBytes, e := json.Marshal(body)
		if e != nil {
			return nil, fmt.Errorf("failed to marshall request body, %w", e)
		}

		inputJSON = ioutil.NopCloser(bytes.NewReader(jsonBytes))

		r.log.V(2).Info("request", "body", string(jsonBytes))
	}

	if query == nil {
		query = &url.Values{}
	}

	url, err := r.url.Parse(path)
	if err != nil {
		return resp, fmt.Errorf("failed parse path %s, %w", path, err)
	}

	url.RawQuery = query.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, *method, url.String(), inputJSON)
	if err != nil {
		return resp, fmt.Errorf("failed create request for %s %s, %w", *method, url, err)
	}

	if r.headerFields != nil {
		for k, v := range *r.headerFields {
			for _, s := range v {
				httpReq.Header.Add(k, s)
			}
		}
	}

	if header != nil {
		for k, v := range *header {
			for _, s := range v {
				httpReq.Header.Add(k, s)
			}
		}
	}

	r.log.V(2).Info("request", "method", *method, "url", url)

	retries := defaultRetries
	sleep := defaultRetryInterval
	start := time.Now()

	for {
		resp, err = r.client.Do(httpReq) // nolint:bodyclose // ok
		if err != nil {                  // nolint:nestif // ok
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "http2: no cached connection was available") ||
				strings.Contains(err.Error(), "net/http: TLS handshake timeout") ||
				strings.Contains(err.Error(), "i/o timeout") ||
				strings.Contains(err.Error(), "unexpected EOF") ||
				strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
				time.Sleep(time.Second * time.Duration(int64(sleep)))

				retries--

				sleep += sleep

				if sleep > backoffResetInterval {
					sleep = defaultRetryInterval
				}

				if retries > 0 || time.Since(start) > *r.timeout {
					r.log.Error(err, "server failed to respond", "method", *method, "url", url)
					continue
				}
			}

			return resp, fmt.Errorf("request %s %s failed, %w", *method, url, err)
		}

		if resp.StatusCode == 404 { //|| (*method == Delete && resp.StatusCode == 204)} {
			return resp, gitprovider.ErrNotFound
		}

		if resp.StatusCode == http.StatusOK || (resp.StatusCode == http.StatusCreated && *method == http.MethodPost) || (resp.StatusCode == http.StatusNoContent && *method == http.MethodDelete) ||
			(resp.StatusCode == http.StatusAccepted && *method == http.MethodDelete) || (resp.StatusCode == http.StatusNoContent && *method == http.MethodPut) {
			return resp, nil
		}

		return resp, fmt.Errorf("request %s %s returned status code: %s, %w\n", *method, url, resp.Status, ErrorUnexpectedStatusCode)
	}
}

// GetRespBody is used to obtain the response body as a string.
func GetRespBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
