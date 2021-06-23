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
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
)

// ChainableRoundTripperFunc is a function that returns a higher-level "out" RoundTripper,
// chained to call the "in" RoundTripper internally, with extra logic. This function must be able
// to handle "in" being nil, and use the http.DefaultTransport default RoundTripper in that case.
// "out" must never be nil.
type ChainableRoundTripperFunc func(in http.RoundTripper) (out http.RoundTripper)

// CommonClientOptions is a struct containing options that are generic to all clients.
type CommonClientOptions struct {
	// Domain specifies the target domain for the client. If unset, the default domain for the
	// given provider will be used (often exposed as DefaultDomain in respective package).
	// The behaviour when setting this flag might vary between providers, read the documentation on
	// NewClient for more information.
	Domain *string

	// EnableDestructiveAPICalls is a flag specifying whether destructive API calls (like
	// deleting a repository) are allowed in the Client. Default: false
	EnableDestructiveAPICalls *bool

	// PreChainRoundTripper is a function to get a custom RoundTripper that is given as the Transport
	// to the *http.Client given to the provider-specific Client. It can be set for doing arbitrary
	// modifications to HTTP requests. "in" might be nil, if so http.DefaultTransport is recommended.
	// The "chain" looks like follows:
	// Git provider API <-> "Post Chain" <-> Provider Specific (e.g. auth, caching) (in) <-> "Pre Chain" (out) <-> *http.Client
	PreChainTransportHook ChainableRoundTripperFunc

	// PostChainTransportHook is a function to get a custom RoundTripper that is the "final" Transport
	// in the chain before talking to the backing API. It can be set for doing arbitrary
	// modifications to HTTP requests. "in" is always nil. It's recommended to internally use http.DefaultTransport.
	// The "chain" looks like follows:
	// Git provider API (in==nil) <-> "Post Chain" (out) <-> Provider Specific (e.g. auth, caching) <-> "Pre Chain" <-> *http.Client
	PostChainTransportHook ChainableRoundTripperFunc

	// Logger allows the caller to pass a logger for use by the provider
	Logger *logr.Logger
}

// ApplyToCommonClientOptions applies the currently set fields in opts to target. If both opts and
// target has the same specific field set, ErrInvalidClientOptions is returned.
func (opts *CommonClientOptions) ApplyToCommonClientOptions(target *CommonClientOptions) error {
	if opts.Domain != nil {
		// Make sure the user didn't specify the Domain twice
		if target.Domain != nil {
			return fmt.Errorf("option Domain already configured: %w", ErrInvalidClientOptions)
		}
		// Don't allow an empty string
		if len(*opts.Domain) == 0 {
			return fmt.Errorf("option Domain cannot be an empty string: %w", ErrInvalidClientOptions)
		}
		target.Domain = opts.Domain
	}

	if opts.EnableDestructiveAPICalls != nil {
		// Make sure the user didn't specify the EnableDestructiveAPICalls twice
		if target.EnableDestructiveAPICalls != nil {
			return fmt.Errorf("option EnableDestructiveAPICalls already configured: %w", ErrInvalidClientOptions)
		}
		target.EnableDestructiveAPICalls = opts.EnableDestructiveAPICalls
	}

	if opts.PreChainTransportHook != nil {
		// Make sure the user didn't specify the PreChainTransportHook twice
		if target.PreChainTransportHook != nil {
			return fmt.Errorf("option PreChainTransportHook already configured: %w", ErrInvalidClientOptions)
		}
		target.PreChainTransportHook = opts.PreChainTransportHook
	}

	if opts.PostChainTransportHook != nil {
		// Make sure the user didn't specify the PostChainTransportHook twice
		if target.PostChainTransportHook != nil {
			return fmt.Errorf("option PostChainTransportHook already configured: %w", ErrInvalidClientOptions)
		}
		target.PostChainTransportHook = opts.PostChainTransportHook
	}

	if opts.Logger != nil {
		if target.Logger != nil {
			return fmt.Errorf("option Logger already configured: %w", ErrInvalidClientOptions)
		}
		target.Logger = opts.Logger
	}
	return nil
}

// BuildClientFromTransportChain builds a *http.Client from a chain of ChainableRoundTripperFuncs.
// The first function in the chain is called with "in" == nil. "out" of the first function in the chain,
// is passed as "in" to the second function, and so on. "out" of the last function in the chain is used
// as net/http Client.Transport.
func BuildClientFromTransportChain(chain []ChainableRoundTripperFunc) (*http.Client, error) {
	var transport http.RoundTripper
	for _, rtFunc := range chain {
		transport = rtFunc(transport)
		if transport == nil {
			return nil, ErrInvalidTransportChainReturn
		}
	}
	return &http.Client{Transport: transport}, nil
}
