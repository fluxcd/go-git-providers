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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"github.com/fluxcd/go-git-providers/gitprovider/cache"
	"github.com/go-logr/logr"
	"golang.org/x/oauth2"
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

	// PreChainTransportHook is a function to get a custom RoundTripper that is given as the Transport
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

	// CABundle is a []byte containing the CA bundle to use for the client.
	CABundle []byte
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

	if opts.CABundle != nil {
		if target.CABundle != nil {
			return fmt.Errorf("option CABundle already configured: %w", ErrInvalidClientOptions)
		}
		target.CABundle = opts.CABundle
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

// ClientOption is the interface to implement for passing options to NewClient.
// The clientOptions struct is private to force usage of the With... functions.
type ClientOption interface {
	// ApplyToClientOptions applies set fields of this object into target.
	ApplyToClientOptions(target *ClientOptions) error
}

// ClientOptions is the struct that tracks data about what options have been set.
type ClientOptions struct {
	// clientOptions shares all the common options
	CommonClientOptions

	// authTransport is a ChainableRoundTripperFunc adding authentication credentials to the transport chain.
	authTransport ChainableRoundTripperFunc

	// enableConditionalRequests will be set if conditional requests should be used.
	enableConditionalRequests *bool
}

// ApplyToClientOptions implements ClientOption, and applies the set fields of opts
// into target. If both opts and target has the same specific field set, ErrInvalidClientOptions is returned.
func (opts *ClientOptions) ApplyToClientOptions(target *ClientOptions) error {
	// Apply common values, if any
	if err := opts.CommonClientOptions.ApplyToCommonClientOptions(&target.CommonClientOptions); err != nil {
		return err
	}

	if opts.authTransport != nil {
		// Make sure the user didn't specify the authTransport twice
		if target.authTransport != nil {
			return fmt.Errorf("option authTransport already configured: %w", ErrInvalidClientOptions)
		}
		target.authTransport = opts.authTransport
	}

	if opts.enableConditionalRequests != nil {
		// Make sure the user didn't specify the enableConditionalRequests twice
		if target.enableConditionalRequests != nil {
			return fmt.Errorf("option enableConditionalRequests already configured: %w", ErrInvalidClientOptions)
		}
		target.enableConditionalRequests = opts.enableConditionalRequests
	}
	return nil
}

// GetTransportChain builds the full chain of transports (from left to right,
// as per gitprovider.BuildClientFromTransportChain) of the form described in NewClient.
func (opts *ClientOptions) GetTransportChain() (chain []ChainableRoundTripperFunc) {
	if opts.PostChainTransportHook != nil {
		chain = append(chain, opts.PostChainTransportHook)
	}
	if opts.authTransport != nil {
		chain = append(chain, opts.authTransport)
	}
	if opts.enableConditionalRequests != nil && *opts.enableConditionalRequests {
		// TODO: Provide some kind of debug logging if/when the httpcache is used
		// One can see if the request hit the cache using: resp.Header[httpcache.XFromCache]
		chain = append(chain, cache.NewHTTPCacheTransport)
	}
	if opts.PreChainTransportHook != nil {
		chain = append(chain, opts.PreChainTransportHook)
	}
	return
}

// buildCommonOption is a helper for returning a ClientOption out of a common option field.
func buildCommonOption(opt CommonClientOptions) *ClientOptions {
	return &ClientOptions{CommonClientOptions: opt}
}

// errorOption implements ClientOption, and just wraps an error which is immediately returned.
// This struct can be used through the optionError function, in order to make makeOptions fail
// if there are invalid options given to the With... functions.
type errorOption struct {
	err error
}

// ApplyToClientOptions implements ClientOption, but just returns the internal error.
func (e *errorOption) ApplyToClientOptions(*ClientOptions) error { return e.err }

// optionError is a constructor for errorOption.
func optionError(err error) ClientOption {
	return &errorOption{err}
}

//
// Common options
//

// WithDomain initializes a Client for a custom instance of the given domain.
// Only host and port information should be present in domain. domain must not be an empty string.
func WithDomain(domain string) ClientOption {
	return buildCommonOption(CommonClientOptions{Domain: &domain})
}

// WithLogger initializes a Client for a custom Stash instance with a logger.
func WithLogger(log *logr.Logger) ClientOption {
	return buildCommonOption(CommonClientOptions{Logger: log})
}

// WithDestructiveAPICalls tells the client whether it's allowed to do dangerous and possibly destructive
// actions, like e.g. deleting a repository.
func WithDestructiveAPICalls(destructiveActions bool) ClientOption {
	return buildCommonOption(CommonClientOptions{EnableDestructiveAPICalls: &destructiveActions})
}

// WithPreChainTransportHook registers a ChainableRoundTripperFunc "before" the cache and authentication
// transports in the chain. For more information, see NewClient, and gitprovider.CommonClientOptions.PreChainTransportHook.
func WithPreChainTransportHook(preRoundTripperFunc ChainableRoundTripperFunc) ClientOption {
	// Don't allow an empty value
	if preRoundTripperFunc == nil {
		return optionError(fmt.Errorf("preRoundTripperFunc cannot be nil: %w", ErrInvalidClientOptions))
	}

	return buildCommonOption(CommonClientOptions{PreChainTransportHook: preRoundTripperFunc})
}

// WithPostChainTransportHook registers a ChainableRoundTripperFunc "after" the cache and authentication
// transports in the chain. For more information, see NewClient, and gitprovider.CommonClientOptions.WithPostChainTransportHook.
func WithPostChainTransportHook(postRoundTripperFunc ChainableRoundTripperFunc) ClientOption {
	// Don't allow an empty value
	if postRoundTripperFunc == nil {
		return optionError(fmt.Errorf("postRoundTripperFunc cannot be nil: %w", ErrInvalidClientOptions))
	}

	return buildCommonOption(CommonClientOptions{PostChainTransportHook: postRoundTripperFunc})
}

// WithOAuth2Token initializes a Client which authenticates with Stash through an OAuth2 token.
// oauth2Token must not be an empty string.
func WithOAuth2Token(oauth2Token string) ClientOption {
	// Don't allow an empty value
	if oauth2Token == "" {
		return optionError(fmt.Errorf("oauth2Token cannot be empty: %w", ErrInvalidClientOptions))
	}

	return &ClientOptions{authTransport: oauth2Transport(oauth2Token)}
}

func oauth2Transport(oauth2Token string) ChainableRoundTripperFunc {
	return func(in http.RoundTripper) http.RoundTripper {
		// Create a TokenSource of the given access token
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: oauth2Token})
		// Create a Transport, with "in" as the underlying transport, and the given TokenSource
		return &oauth2.Transport{
			Base:   in,
			Source: oauth2.ReuseTokenSource(nil, ts),
		}
	}
}

// WithConditionalRequests instructs the client to use Conditional Requests to Stash.
// See: https://gitlab.com/gitlab.org/gitlab.foss/-/issues/26926, and
// https://docs.gitlab.com/ee/development/polling.html for more info.
func WithConditionalRequests(conditionalRequests bool) ClientOption {
	return &ClientOptions{enableConditionalRequests: &conditionalRequests}
}

// MakeClientOptions assembles a clientOptions struct from ClientOption mutator functions.
func MakeClientOptions(opts ...ClientOption) (*ClientOptions, error) {
	o := &ClientOptions{}
	for _, opt := range opts {
		if err := opt.ApplyToClientOptions(o); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// WithCustomCAPostChainTransportHook registers a ChainableRoundTripperFunc "after" the cache and authentication
// transports in the chain.
func WithCustomCAPostChainTransportHook(caBundle []byte) ClientOption {
	// Don't allow an empty value
	if len(caBundle) == 0 {
		return optionError(fmt.Errorf("caBundle cannot be empty: %w", ErrInvalidClientOptions))
	}

	return buildCommonOption(CommonClientOptions{CABundle: caBundle, PostChainTransportHook: caCustomTransport(caBundle)})
}

func caCustomTransport(caBundle []byte) ChainableRoundTripperFunc {
	return func(_ http.RoundTripper) http.RoundTripper {
		// discard error, as we're only using it to check if rootCA is empty
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		rootCAs.AppendCertsFromPEM(caBundle)

		return &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: rootCAs,
			},
		}
	}
}
