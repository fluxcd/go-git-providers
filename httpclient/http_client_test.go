package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

func Test_NewRequester(t *testing.T) {
	testCases := []struct {
		name      string
		domain    string
		header    *http.Header
		timeout   *time.Duration
		client    *http.Client
		transport http.RoundTripper
		log       logr.Logger
		output    string
	}{
		{
			name:      "no domain",
			domain:    "",
			header:    &http.Header{},
			timeout:   &DefaultTimeout,
			client:    &http.Client{},
			transport: defaultTransport,
			log:       logr.Logger{},
			output:    "domain is required",
		},
		{
			name:      "wrong domain",
			domain:    "local#.@*dev",
			header:    &http.Header{},
			timeout:   &DefaultTimeout,
			client:    &http.Client{},
			transport: defaultTransport,
			log:       logr.Logger{},
			output:    `failed to parse domain to url, parse "https://local#.@*dev": net/url: invalid userinfo`,
		},
		{
			name:   "localhost domain",
			domain: "http://localhost",
			output: "http://localhost",
		}, {
			name:      "my specific domain",
			domain:    "my.domain",
			header:    &http.Header{},
			timeout:   &DefaultTimeout,
			client:    &http.Client{Transport: defaultTransport},
			transport: defaultTransport,
			log:       initLogger(),
			output:    "https://my.domain",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var val string
			r, err := NewRequester(tc.domain, tc.header, tc.timeout, tc.client, tc.transport, tc.log)
			if err != nil {
				val = fmt.Sprintf("%s", err)
			}
			if r != nil {
				val = r.(*reqResp).url.String()
			}
			if val != tc.output {
				t.Errorf("Expected %s, got %s", tc.output, val)
			}
		})
	}
}

func Test_Do(t *testing.T) {
	type user struct {
		Name  []string `json:"name"`
		Email []string `json:"email"`
	}

	testCases := []struct {
		name   string
		path   string
		query  *url.Values
		method string
		body   interface{}
		header *http.Header
		output interface{}
	}{
		{
			name: "test GET method",
			path: "users",
			query: &url.Values{
				"name":  []string{"john", "doe"},
				"email": []string{"jdoe@programmer.net"},
			},
			method: "GET",
			output: user{
				Name:  []string{"john", "doe"},
				Email: []string{"jdoe@programmer.net"},
			},
		}, {
			name:   "test POST method",
			path:   "users",
			method: "POST",
			body: &user{
				Name:  []string{"tony", "stark"},
				Email: []string{"tony@stark.entreprise"},
			},
			output: "201 - OK!",
		},
		{
			name:   "test POST without body",
			path:   "users",
			method: "POST",
			output: "201 - OK!",
		},
	}

	// declare a Requester
	r := reqResp{
		transport:    defaultTransport,
		log:          initLogger(),
		timeout:      &DefaultTimeout,
		headerFields: &http.Header{},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Start a local HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var resp []byte
				switch req.Method {
				case "GET":
					response, err := json.Marshal(tc.query)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						resp = []byte("500 - Failed to retrieve data!")
					} else {
						w.WriteHeader(http.StatusOK)
						resp = response
					}
				case "POST":
					w.WriteHeader(http.StatusCreated)
					resp = []byte("201 - OK!")
				case "PUT":
					w.WriteHeader(http.StatusCreated)
					resp = []byte("201 - OK!")
				}
				w.Write(resp)
			}))

			// Close the server when test finishes
			defer server.Close()

			url, _ := url.ParseRequestURI(server.URL)
			// tie Requester url and client to the fake server
			r.url = url
			r.client = server.Client()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			res, _ := r.Do(ctx, tc.path, tc.query, tc.body, &tc.method, tc.header)

			if tc.method == "GET" {
				user := user{}
				err := json.NewDecoder(res.Body).Decode(&user)
				if err != nil {
					t.Fatalf("%s users failed, unable to obtain response body: %v", tc.method, err)
				}
				if !reflect.DeepEqual(user, tc.output) {
					t.Errorf("Expected %s, got %s", tc.output, user)
				}
			} else {
				r, err := GetRespBody(res)
				if err != nil {
					t.Fatalf("%s users failed, unable to obtain response body: %v", tc.method, err)
				}
				if !reflect.DeepEqual(r, tc.output) {
					t.Errorf("Expected %s, got %s", tc.output, r)
				}
			}
		})
	}
}

func initLogger() logr.Logger {
	var log logr.Logger
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("impossible to initialize a zap logger (%v)?", err))
	}
	log = zapr.NewLogger(zapLog)
	return log
}
