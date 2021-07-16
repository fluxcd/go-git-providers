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

package stash

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/gitprovider/cache"
	"github.com/fluxcd/go-git-providers/validation"
)

func dummyRoundTripper1(http.RoundTripper) http.RoundTripper { return nil }
func dummyRoundTripper2(http.RoundTripper) http.RoundTripper { return nil }
func dummyRoundTripper3(http.RoundTripper) http.RoundTripper { return nil }

func roundTrippersEqual(a, b gitprovider.ChainableRoundTripperFunc) bool {
	if a == nil && b == nil {
		return true
	} else if (a != nil && b == nil) || (a == nil && b != nil) {
		return false
	}
	// Note that this comparison relies on "undefined behavior" in the Go language spec, see:
	// https://stackoverflow.com/questions/9643205/how-do-i-compare-two-functions-for-pointer-equality-in-the-latest-go-weekly
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

func Test_clientOptions_getTransportChain(t *testing.T) {
	tests := []struct {
		name      string
		preChain  gitprovider.ChainableRoundTripperFunc
		postChain gitprovider.ChainableRoundTripperFunc
		auth      gitprovider.ChainableRoundTripperFunc
		cache     bool
		wantChain []gitprovider.ChainableRoundTripperFunc
	}{
		{
			name:      "all roundtrippers",
			preChain:  dummyRoundTripper1,
			postChain: dummyRoundTripper2,
			auth:      dummyRoundTripper3,
			cache:     true,
			// expect: "post chain" <-> "auth" <-> "cache" <-> "pre chain"
			wantChain: []gitprovider.ChainableRoundTripperFunc{
				dummyRoundTripper2,
				dummyRoundTripper3,
				cache.NewHTTPCacheTransport,
				dummyRoundTripper1,
			},
		},
		{
			name:     "only pre + auth",
			preChain: dummyRoundTripper1,
			auth:     dummyRoundTripper2,
			// expect: "auth" <-> "pre chain"
			wantChain: []gitprovider.ChainableRoundTripperFunc{
				dummyRoundTripper2,
				dummyRoundTripper1,
			},
		},
		{
			name:  "only cache + auth",
			cache: true,
			auth:  dummyRoundTripper1,
			// expect: "auth" <-> "cache"
			wantChain: []gitprovider.ChainableRoundTripperFunc{
				dummyRoundTripper1,
				cache.NewHTTPCacheTransport,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &clientOptions{
				CommonClientOptions: gitprovider.CommonClientOptions{
					PreChainTransportHook:  tt.preChain,
					PostChainTransportHook: tt.postChain,
				},
				AuthTransport: tt.auth,
			}
			gotChain := opts.getTransportChain()
			for i := range tt.wantChain {
				if !roundTrippersEqual(tt.wantChain[i], gotChain[i]) {
					t.Errorf("clientOptions.getTransportChain() = %v, want %v", gotChain, tt.wantChain)
				}
				break
			}
		})
	}
}

func Test_makeOptions(t *testing.T) {
	tests := []struct {
		name         string
		opts         []ClientOption
		want         *clientOptions
		expectedErrs []error
	}{
		{
			name: "no options",
			want: &clientOptions{},
		},
		{
			name: "WithDomain",
			opts: []ClientOption{WithDomain("foo")},
			want: buildCommonOption(gitprovider.CommonClientOptions{Domain: gitprovider.StringVar("foo")}),
		},
		{
			name:         "WithDomain, empty",
			opts:         []ClientOption{WithDomain("")},
			expectedErrs: []error{gitprovider.ErrInvalidClientOptions},
		},
		{
			name: "WithDestructiveAPICalls",
			opts: []ClientOption{WithDestructiveAPICalls(true)},
			want: buildCommonOption(gitprovider.CommonClientOptions{EnableDestructiveAPICalls: gitprovider.BoolVar(true)}),
		},
		{
			name: "WithPreChainTransportHook",
			opts: []ClientOption{WithPreChainTransportHook(dummyRoundTripper1)},
			want: buildCommonOption(gitprovider.CommonClientOptions{PreChainTransportHook: dummyRoundTripper1}),
		},
		{
			name:         "WithPreChainTransportHook, nil",
			opts:         []ClientOption{WithPreChainTransportHook(nil)},
			expectedErrs: []error{gitprovider.ErrInvalidClientOptions},
		},
		{
			name: "WithPostChainTransportHook",
			opts: []ClientOption{WithPostChainTransportHook(dummyRoundTripper2)},
			want: buildCommonOption(gitprovider.CommonClientOptions{PostChainTransportHook: dummyRoundTripper2}),
		},
		{
			name:         "WithPostChainTransportHook, nil",
			opts:         []ClientOption{WithPostChainTransportHook(nil)},
			expectedErrs: []error{gitprovider.ErrInvalidClientOptions},
		},
		{
			name: "WithOAuth2Token",
			opts: []ClientOption{WithOAuth2Token("foo")},
			want: &clientOptions{AuthTransport: oauth2Transport("foo")},
		},
		{
			name:         "WithOAuth2Token, empty",
			opts:         []ClientOption{WithOAuth2Token("")},
			expectedErrs: []error{gitprovider.ErrInvalidClientOptions},
		},
		{
			name: "WithConditionalRequests",
			opts: []ClientOption{WithConditionalRequests(true)},
			want: &clientOptions{EnableConditionalRequests: gitprovider.BoolVar(true)},
		},
		{
			name:         "WithConditionalRequests, exclusive",
			opts:         []ClientOption{WithConditionalRequests(true), WithConditionalRequests(false)},
			expectedErrs: []error{gitprovider.ErrInvalidClientOptions},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeOptions(tt.opts...)
			validation.TestExpectErrors(t, "makeOptions", err, tt.expectedErrs...)
			if tt.want == nil {
				return
			}
			if !roundTrippersEqual(got.AuthTransport, tt.want.AuthTransport) ||
				!roundTrippersEqual(got.PostChainTransportHook, tt.want.PostChainTransportHook) ||
				!roundTrippersEqual(got.PreChainTransportHook, tt.want.PreChainTransportHook) {
				t.Errorf("makeOptions() = %v, want %v", got, tt.want)
			}
			got.AuthTransport = nil
			got.PostChainTransportHook = nil
			got.PreChainTransportHook = nil
			tt.want.AuthTransport = nil
			tt.want.PostChainTransportHook = nil
			tt.want.PreChainTransportHook = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
