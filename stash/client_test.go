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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap/zaptest"
)

func Test_NewClient(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		header    *http.Header
		timeout   time.Duration
		client    *http.Client
		transport http.RoundTripper
		log       logr.Logger
		output    string
	}{
		{
			name:      "no host",
			host:      "",
			header:    &http.Header{},
			timeout:   defaultTimeout,
			client:    &http.Client{},
			transport: defaultTransport,
			log:       logr.Discard(),
			output:    "host is required",
		},
		{
			name:      "wrong host",
			host:      "local#.@*dev",
			header:    &http.Header{},
			timeout:   defaultTimeout,
			client:    &http.Client{},
			transport: defaultTransport,
			log:       logr.Discard(),
			output:    `failed to parse host https://local#.@*dev to url, parse "https://local#.@*dev": net/url: invalid userinfo`,
		},
		{
			name:   "default host",
			log:    logr.Discard(),
			host:   fmt.Sprint("http://" + defaultHost),
			output: fmt.Sprint("http://" + defaultHost),
		}, {
			name:      "my specific host",
			host:      "my.host",
			header:    &http.Header{},
			timeout:   defaultTimeout,
			client:    &http.Client{Transport: defaultTransport},
			transport: defaultTransport,
			log:       initLogger(t),
			output:    "https://my.host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var val string
			c, err := NewClient(&http.Client{}, tt.host, tt.header, tt.log, func(c *Client) error {
				c.Client.HTTPClient = &http.Client{
					Transport: tt.transport,
					Timeout:   tt.timeout,
				}
				return nil
			})
			if err != nil {
				val = fmt.Sprintf("%s", err)
			}
			if c != nil {
				val = c.BaseURL.String()
			}
			if val != tt.output {
				t.Errorf("Expected %s, got %s", tt.output, val)
			}
		})
	}
}

func Test_Do(t *testing.T) {
	type user struct {
		Name  []string `json:"name"`
		Email []string `json:"email"`
	}

	tests := []struct {
		name   string
		path   string
		query  url.Values
		method string
		body   interface{}
		header http.Header
		output interface{}
	}{
		{
			name: "test GET method",
			path: "users",
			query: url.Values{
				"name":  []string{"john", "doe"},
				"email": []string{"jdoe@programmer.net"},
			},
			method: http.MethodGet,
			output: user{
				Name:  []string{"john", "doe"},
				Email: []string{"jdoe@programmer.net"},
			},
		}, {
			name:   "test POST method",
			path:   "users",
			method: http.MethodPost,
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			body: &user{
				Name:  []string{"tony", "stark"},
				Email: []string{"tony@stark.entreprise"},
			},
			output: []byte("201 - OK!"),
		},
		{
			name:   "test POST with wrong headers method",
			path:   "users",
			method: http.MethodPost,
			header: http.Header{
				"Content-Type": []string{"text/html"},
			},
			body: &user{
				Name:  []string{"tony", "stark"},
				Email: []string{"tony@stark.entreprise"},
			},
			output: []byte("400 - NOK!"),
		},
		{
			name:   "test POST without body",
			path:   "users",
			method: http.MethodPost,
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			output: []byte("201 - OK!"),
		},
	}

	berearHeader := &http.Header{
		"WWW-Authenticate": []string{"Bearer"},
	}

	// declare a Client
	c, err := NewClient(nil, defaultHost, berearHeader, initLogger(t))
	if err != nil {
		t.Fatalf("unexpected error while declaring a client: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start a local HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var resp []byte
				if ctype := req.Header.Get("WWW-Authenticate"); ctype != "Bearer" {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte("400 - NOK!"))
					return
				}
				switch req.Method {
				case http.MethodGet:
					response, err := json.Marshal(tt.query)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						resp = []byte("500 - Failed to retrieve data!")
					} else {
						w.WriteHeader(http.StatusOK)
						resp = response
					}
				case http.MethodPost:
					if ctype := req.Header.Get("Content-Type"); ctype != "application/json" {
						w.WriteHeader(http.StatusBadRequest)
						resp = []byte("400 - NOK!")
					} else {
						w.WriteHeader(http.StatusCreated)
						resp = []byte("201 - OK!")
					}
				case http.MethodPut:
					w.WriteHeader(http.StatusCreated)
					resp = []byte("201 - OK!")
				}
				w.Write(resp)
			}))

			url, _ := url.ParseRequestURI(server.URL)
			// tie Requester url and client to the fake server
			c.BaseURL = url
			c.Client.HTTPClient = server.Client()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var bodyReader io.ReadCloser
			jsonBody, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatalf("failed to marshall request body: %v", err)
			}

			bodyReader = io.NopCloser(bytes.NewReader(jsonBody))
			req, err := c.NewRequest(ctx, tt.method, tt.path,
				WithQuery(tt.query),
				WithBody(bodyReader),
				WithHeader(tt.header))

			if err != nil {
				t.Fatalf("request generation failed with error: %v", err)
			}

			res, _, err := c.Do(req)
			if err != nil {
				t.Fatalf("request failed with error: %v", err)
			}

			if tt.method == http.MethodGet {
				user := user{}
				err := json.Unmarshal(res, &user)
				if err != nil {
					t.Fatalf("%s users failed, unable to obtain response body: %v", tt.method, err)
				}
				if !reflect.DeepEqual(user, tt.output) {
					t.Errorf("Expected %s, got %s", tt.output, user)
				}
			} else {
				if err != nil {
					t.Fatalf("%s users failed, unable to obtain response body: %v", tt.method, err)
				}
				if !reflect.DeepEqual(res, tt.output) {
					t.Errorf("Expected %s, got %s", tt.output, res)
				}
			}
		})
	}
}

func Test_DoWithRetry(t *testing.T) {
	tests := []struct {
		name               string
		retries            int
		retryMin, retryMax time.Duration
		output             []byte
	}{
		{
			name:     "test 2 retries",
			retries:  2,
			retryMin: 1 * time.Millisecond,
			retryMax: 2 * time.Millisecond,
			output:   []byte("2"),
		},
		{
			name:     "test 5 retries",
			retries:  5,
			retryMin: 1 * time.Millisecond,
			retryMax: 2 * time.Millisecond,
			output:   []byte("5"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start a local HTTP server
			retries := 1
			c := NewTestClient(t, func(req *http.Request) (*http.Response, error) {
				if retries < tt.retries {
					retries++
					return nil, fmt.Errorf("connection refused, please retry")
				}
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprint(retries))),
					Header:     make(http.Header),
				}, nil
			}, func(c *Client) error {
				c.Client.RetryWaitMin = tt.retryMin
				c.Client.RetryWaitMax = tt.retryMax
				c.Client.RetryMax = tt.retries
				return nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			request, err := c.NewRequest(ctx, http.MethodGet, "")
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			res, _, err := c.Do(request)
			if err != nil {
				if !strings.Contains(err.Error(), string(tt.output)) {
					t.Fatalf("request failed: %v", err)
				}
				return
			}

			if !reflect.DeepEqual(res, tt.output) {
				t.Errorf("Expected %s, got %s", tt.output, res)
			}
		})
	}
}

func initLogger(t *testing.T) logr.Logger {
	var log logr.Logger
	zapLog := zaptest.NewLogger(t)
	log = zapr.NewLogger(zapLog)
	return log
}

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) (*http.Response, error)

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

//NewTestClient returns a Client with Transport replaced to avoid making real calls
func NewTestClient(t *testing.T, fn RoundTripFunc, opts ...ClientOptionsFunc) *Client {
	c, err := NewClient(nil, defaultHost, nil, initLogger(t))
	if err != nil {
		t.Fatalf("unexpected error while declaring a client: %v", err)
	}

	c.Client.HTTPClient.Transport = RoundTripFunc(fn)

	for _, opt := range opts {
		opt(c)
	}

	return c
}
