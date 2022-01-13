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

package gitprovider

import (
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/fluxcd/go-git-providers/validation"
)

func dummyRoundTripper1(http.RoundTripper) http.RoundTripper { return nil }
func dummyRoundTripper2(http.RoundTripper) http.RoundTripper { return nil }
func dummyRoundTripper3(http.RoundTripper) http.RoundTripper { return nil }

func roundTrippersEqual(a, b ChainableRoundTripperFunc) bool {
	if a == nil && b == nil {
		return true
	} else if (a != nil && b == nil) || (a == nil && b != nil) {
		return false
	}
	// Note that this comparison relies on "undefined behavior" in the Go language spec, see:
	// https://stackoverflow.com/questions/9643205/how-do-i-compare-two-functions-for-pointer-equality-in-the-latest-go-weekly
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

type commonClientOption interface {
	ApplyToCommonClientOptions(*CommonClientOptions) error
}

func makeOptions(opts ...commonClientOption) (*CommonClientOptions, error) {
	o := &CommonClientOptions{}
	for _, opt := range opts {
		if err := opt.ApplyToCommonClientOptions(o); err != nil {
			return nil, err
		}
	}
	return o, nil
}

func withDomain(domain string) commonClientOption {
	return &CommonClientOptions{Domain: &domain}
}

func withDestructiveAPICalls(destructiveActions bool) commonClientOption {
	return &CommonClientOptions{EnableDestructiveAPICalls: &destructiveActions}
}

func withPreChainTransportHook(preRoundTripperFunc ChainableRoundTripperFunc) commonClientOption {
	return &CommonClientOptions{PreChainTransportHook: preRoundTripperFunc}
}

func withPostChainTransportHook(postRoundTripperFunc ChainableRoundTripperFunc) commonClientOption {
	return &CommonClientOptions{PostChainTransportHook: postRoundTripperFunc}
}

func Test_makeOptions(t *testing.T) {
	tests := []struct {
		name         string
		opts         []commonClientOption
		want         *CommonClientOptions
		expectedErrs []error
	}{
		{
			name: "no options",
			want: &CommonClientOptions{},
		},
		{
			name: "withDomain",
			opts: []commonClientOption{withDomain("foo")},
			want: &CommonClientOptions{Domain: StringVar("foo")},
		},
		{
			name:         "withDomain, empty",
			opts:         []commonClientOption{withDomain("")},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name:         "withDomain, duplicate",
			opts:         []commonClientOption{withDomain("foo"), withDomain("bar")},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "withDestructiveAPICalls",
			opts: []commonClientOption{withDestructiveAPICalls(true)},
			want: &CommonClientOptions{EnableDestructiveAPICalls: BoolVar(true)},
		},
		{
			name:         "withDestructiveAPICalls, duplicate",
			opts:         []commonClientOption{withDestructiveAPICalls(true), withDestructiveAPICalls(false)},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "withPreChainTransportHook",
			opts: []commonClientOption{withPreChainTransportHook(dummyRoundTripper1)},
			want: &CommonClientOptions{PreChainTransportHook: dummyRoundTripper1},
		},
		{
			name:         "withPreChainTransportHook, duplicate",
			opts:         []commonClientOption{withPreChainTransportHook(dummyRoundTripper1), withPreChainTransportHook(dummyRoundTripper1)},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "withPostChainTransportHook",
			opts: []commonClientOption{withPostChainTransportHook(dummyRoundTripper1)},
			want: &CommonClientOptions{PostChainTransportHook: dummyRoundTripper1},
		},
		{
			name:         "withPostChainTransportHook, duplicate",
			opts:         []commonClientOption{withPostChainTransportHook(dummyRoundTripper1), withPostChainTransportHook(dummyRoundTripper1)},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeOptions(tt.opts...)
			validation.TestExpectErrors(t, "makeOptions", err, tt.expectedErrs...)
			if tt.want == nil {
				return
			}
			if !roundTrippersEqual(got.PostChainTransportHook, tt.want.PostChainTransportHook) ||
				!roundTrippersEqual(got.PreChainTransportHook, tt.want.PreChainTransportHook) {
				t.Errorf("makeOptions() = %v, want %v", got, tt.want)
			}
			got.PostChainTransportHook = nil
			got.PreChainTransportHook = nil
			tt.want.PostChainTransportHook = nil
			tt.want.PreChainTransportHook = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientOptions_getTransportChain(t *testing.T) {
	tests := []struct {
		name      string
		preChain  ChainableRoundTripperFunc
		postChain ChainableRoundTripperFunc
		auth      ChainableRoundTripperFunc
		cache     bool
		wantChain []ChainableRoundTripperFunc
	}{
		{
			name:      "all roundtrippers",
			preChain:  dummyRoundTripper1,
			postChain: dummyRoundTripper2,
			auth:      dummyRoundTripper3,
			cache:     true,
			// expect: "post chain" <-> "auth" <-> "cache" <-> "pre chain"
			wantChain: []ChainableRoundTripperFunc{
				dummyRoundTripper2,
				dummyRoundTripper3,
				dummyRoundTripper1,
			},
		},
		{
			name:     "only pre + auth",
			preChain: dummyRoundTripper1,
			auth:     dummyRoundTripper2,
			// expect: "auth" <-> "pre chain"
			wantChain: []ChainableRoundTripperFunc{
				dummyRoundTripper2,
				dummyRoundTripper1,
			},
		},
		{
			name:  "only cache + auth",
			cache: true,
			auth:  dummyRoundTripper1,
			// expect: "auth" <-> "cache"
			wantChain: []ChainableRoundTripperFunc{
				dummyRoundTripper1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dummy := "dummy"
			opts := &ClientOptions{
				CommonClientOptions: CommonClientOptions{
					Domain:                 &dummy,
					PreChainTransportHook:  tt.preChain,
					PostChainTransportHook: tt.postChain,
				},
				authTransport: tt.auth,
			}
			gotChain := opts.GetTransportChain()
			for i := range tt.wantChain {
				if !roundTrippersEqual(tt.wantChain[i], gotChain[i]) {
					t.Fatalf("%s - clientOptions.getTransportChain() = %v, want %v", tt.name, gotChain, tt.wantChain)
				}
			}
		})
	}
}

func Test_makeCientOptions(t *testing.T) {
	ca, err := os.ReadFile("./testdata/ca.pem")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		opts         []ClientOption
		want         *ClientOptions
		expectedErrs []error
	}{
		{
			name: "no options",
			want: &ClientOptions{},
		},
		{
			name: "WithDomain",
			opts: []ClientOption{WithDomain("foo")},
			want: buildCommonOption(CommonClientOptions{Domain: StringVar("foo")}),
		},
		{
			name:         "WithDomain, empty",
			opts:         []ClientOption{WithDomain("")},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "WithDestructiveAPICalls",
			opts: []ClientOption{WithDestructiveAPICalls(true)},
			want: buildCommonOption(CommonClientOptions{EnableDestructiveAPICalls: BoolVar(true)}),
		},
		{
			name: "WithPreChainTransportHook",
			opts: []ClientOption{WithPreChainTransportHook(dummyRoundTripper1)},
			want: buildCommonOption(CommonClientOptions{PreChainTransportHook: dummyRoundTripper1}),
		},
		{
			name:         "WithPreChainTransportHook, nil",
			opts:         []ClientOption{WithPreChainTransportHook(nil)},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "WithPostChainTransportHook",
			opts: []ClientOption{WithPostChainTransportHook(dummyRoundTripper2)},
			want: buildCommonOption(CommonClientOptions{PostChainTransportHook: dummyRoundTripper2}),
		},
		{
			name:         "WithPostChainTransportHook, nil",
			opts:         []ClientOption{WithPostChainTransportHook(nil)},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "WithCustomCAPostChainTransportHook",
			opts: []ClientOption{WithCustomCAPostChainTransportHook(ca)},
			want: buildCommonOption(CommonClientOptions{CABundle: ca, PostChainTransportHook: caCustomTransport(ca)}),
		},
		{
			name:         "WithCustomCAPostChainTransportHook, nil",
			opts:         []ClientOption{WithCustomCAPostChainTransportHook(nil)},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "WithOAuth2Token",
			opts: []ClientOption{WithOAuth2Token("foo")},
			want: &ClientOptions{authTransport: oauth2Transport("foo")},
		},
		{
			name:         "WithOAuth2Token, empty",
			opts:         []ClientOption{WithOAuth2Token("")},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
		{
			name: "WithConditionalRequests",
			opts: []ClientOption{WithConditionalRequests(true)},
			want: &ClientOptions{enableConditionalRequests: BoolVar(true)},
		},
		{
			name:         "WithConditionalRequests, exclusive",
			opts:         []ClientOption{WithConditionalRequests(true), WithConditionalRequests(false)},
			expectedErrs: []error{ErrInvalidClientOptions},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeClientOptions(tt.opts...)
			validation.TestExpectErrors(t, "makeOptions", err, tt.expectedErrs...)
			if tt.want == nil {
				return
			}
			if !roundTrippersEqual(got.authTransport, tt.want.authTransport) ||
				!roundTrippersEqual(got.PostChainTransportHook, tt.want.PostChainTransportHook) ||
				!roundTrippersEqual(got.PreChainTransportHook, tt.want.PreChainTransportHook) {
				t.Errorf("makeOptions() = %v, want %v", got, tt.want)
			}
			got.authTransport = nil
			got.PostChainTransportHook = nil
			got.PreChainTransportHook = nil
			tt.want.authTransport = nil
			tt.want.PostChainTransportHook = nil
			tt.want.PreChainTransportHook = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
