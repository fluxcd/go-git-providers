/*
Copyright 2020 The Flux CD contributors.

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

package github

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/google/go-github/v45/github"
)

func Test_validateAPIObject(t *testing.T) {
	tests := []struct {
		name         string
		structName   string
		fn           func(validation.Validator)
		expectedErrs []error
	}{
		{
			name:       "no error => nil",
			structName: "Foo",
			fn:         func(validation.Validator) {},
		},
		{
			name:       "one error => MultiError & InvalidServerData",
			structName: "Foo",
			fn: func(v validation.Validator) {
				v.Required("FieldBar")
			},
			expectedErrs: []error{gitprovider.ErrInvalidServerData, &validation.MultiError{}, validation.ErrFieldRequired},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIObject(tt.structName, tt.fn)
			validation.TestExpectErrors(t, "validateAPIObject", err, tt.expectedErrs...)
		})
	}
}

func newGHError() *github.ErrorResponse {
	return &github.ErrorResponse{
		Response: &http.Response{
			Request: &http.Request{
				Method: "GET",
				URL:    &url.URL{},
			},
			StatusCode: 404,
		},
	}
}

func Test_allPages(t *testing.T) {
	tests := []struct {
		name          string
		opts          *github.ListOptions
		fn            func(int) (*github.Response, error)
		expectedErrs  []error
		expectedCalls int
	}{
		{
			name: "one page only, no error",
			opts: &github.ListOptions{},
			fn: func(_ int) (*github.Response, error) {
				return &github.Response{NextPage: 0}, nil
			},
			expectedCalls: 1,
		},
		{
			name: "two pages, no error",
			opts: &github.ListOptions{},
			fn: func(i int) (*github.Response, error) {
				switch i {
				case 1:
					return &github.Response{NextPage: 2}, nil
				}
				return &github.Response{NextPage: 0}, nil
			},
			expectedCalls: 2,
		},
		{
			name: "four pages, error at second",
			opts: &github.ListOptions{},
			fn: func(i int) (*github.Response, error) {
				switch i {
				case 1:
					return &github.Response{NextPage: 2}, nil
				case 2:
					return nil, newGHError()
				case 3:
					return &github.Response{NextPage: 4}, nil
				}
				return &github.Response{NextPage: 0}, nil
			},
			expectedCalls: 2,
			expectedErrs:  []error{&validation.MultiError{}, gitprovider.ErrNotFound, newGHError()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := 0
			// the page index are 1-based, and omitting page is the same as page=1
			// set page=1 here just to be able to test more easily
			tt.opts.Page = 1
			err := allPages(tt.opts, func() (*github.Response, error) {
				i++
				if tt.opts.Page != i {
					t.Fatalf("page number is unexpected: got = %d want = %d", tt.opts.Page, i)
				}
				return tt.fn(i)
			})
			validation.TestExpectErrors(t, "allPages", err, tt.expectedErrs...)
			if i != tt.expectedCalls {
				t.Errorf("allPages() expectedCalls = %v, want %v", i, tt.expectedCalls)
			}
		})
	}
}
