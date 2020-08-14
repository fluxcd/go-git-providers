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
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v32/github"
	"github.com/gregjones/httpcache"
	"golang.org/x/oauth2"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const (
	// defaultDomain specifies the default domain used as the backend.
	defaultDomain = "github.com"
	// patUsername is the "username" for the basic auth authentication flow
	// when using a personal access token as the "password". This string could
	// be arbitrary, even unset, as it is not respected server-side. For conventions'
	// sake, we'll set this to "git".
	patUsername = "git"
)

var (
	// ErrInvalidClientOptions is the error returned when calling NewClient() with
	// invalid options (e.g. specifying mutually exclusive options).
	ErrInvalidClientOptions = errors.New("invalid options given to NewClient()")
	// ErrDestructiveCallDisallowed happens when the client isn't set up with WithDestructiveAPICalls()
	// but a destructive action is called.
	ErrDestructiveCallDisallowed = errors.New("a destructive call was blocked because it wasn't allowed by the client")
)

// clientOptions is the struct that tracks data about what options have been set
// It is private so that the user must use the With... functions.
type clientOptions struct {
	// Domain specifies the backing domain, which can be arbitrary if the user uses
	// GitHub Enterprise. If unset, defaultDomain will be used.
	Domain *string

	// authTransportFactory is a way to acquire a http.RoundTripper with auth credentials configured.
	authTransportFactory authTransportFactory

	// RoundTripperFactory is a factory to get a http.RoundTripper that is sitting between the *github.Client's
	// internal *http.Client, and the *httpcache.Transport RoundTripper. It can be set
	// for doing arbitrary modifications to http requests.
	RoundTripperFactory RoundTripperFactory

	// EnableDestructiveAPICalls is a flag to tell whether destructive API calls like
	// deleting a repository and such is allowed. Default: false
	EnableDestructiveAPICalls *bool

	// EnableConditionalRequests will be set if conditional requests should be used.
	// See: https://developer.github.com/v3/#conditional-requests for more info.
	// Default: false
	EnableConditionalRequests *bool
}

// ClientOption is a function that is mutating a pointer to a clientOptions object
// which holds information of how the Client should be initialized.
type ClientOption func(*clientOptions) error

// WithOAuth2Token initializes a Client which authenticates with GitHub through an OAuth2 token.
// oauth2Token must not be an empty string.
// WithOAuth2Token is mutually exclusive with WithPersonalAccessToken.
func WithOAuth2Token(oauth2Token string) ClientOption {
	return func(opts *clientOptions) error {
		// Don't allow an empty value
		if len(oauth2Token) == 0 {
			return fmt.Errorf("oauth2Token cannot be empty: %w", ErrInvalidClientOptions)
		}
		// Make sure the user didn't specify auth twice
		if opts.authTransportFactory != nil {
			return fmt.Errorf("authentication http.Client already configured: %w", ErrInvalidClientOptions)
		}

		opts.authTransportFactory = &oauth2Auth{oauth2Token}
		return nil
	}
}

// WithPersonalAccessToken initializes a Client which authenticates with GitHub through a personal access token.
// patToken must not be an empty string.
// WithPersonalAccessToken is mutually exclusive with WithOAuth2Token.
func WithPersonalAccessToken(patToken string) ClientOption {
	return func(opts *clientOptions) error {
		// Don't allow an empty value
		if len(patToken) == 0 {
			return fmt.Errorf("patToken cannot be empty: %w", ErrInvalidClientOptions)
		}
		// Make sure the user didn't specify auth twice
		if opts.authTransportFactory != nil {
			return fmt.Errorf("authentication http.Client already configured: %w", ErrInvalidClientOptions)
		}
		opts.authTransportFactory = &patAuth{patToken}
		return nil
	}
}

// WithRoundTripper initializes a Client with a given authTransportFactory, used for acquiring the *http.Client later.
// authTransportFactory must not be nil.
func WithRoundTripper(roundTripper RoundTripperFactory) ClientOption {
	return func(opts *clientOptions) error {
		// Don't allow an empty value
		if roundTripper == nil {
			return fmt.Errorf("roundTripper cannot be nil: %w", ErrInvalidClientOptions)
		}
		// Make sure the user didn't specify the RoundTripperFactory twice
		if opts.RoundTripperFactory != nil {
			return fmt.Errorf("roundTripper already configured: %w", ErrInvalidClientOptions)
		}
		opts.RoundTripperFactory = roundTripper
		return nil
	}
}

// WithDomain initializes a Client for a custom GitHub Enterprise instance of the given domain.
// Only host and port information should be present in domain. domain must not be an empty string.
func WithDomain(domain string) ClientOption {
	return func(opts *clientOptions) error {
		// Don't set an empty value
		if len(domain) == 0 {
			return fmt.Errorf("domain cannot be empty: %w", ErrInvalidClientOptions)
		}
		// Make sure the user didn't specify the domain twice
		if opts.Domain != nil {
			return fmt.Errorf("domain already configured: %w", ErrInvalidClientOptions)
		}
		opts.Domain = gitprovider.StringVar(domain)
		return nil
	}
}

// WithDestructiveAPICalls tells the client whether it's allowed to do dangerous and possibly destructive
// actions, like e.g. deleting a repository.
func WithDestructiveAPICalls(destructiveActions bool) ClientOption {
	return func(opts *clientOptions) error {
		// Make sure the user didn't specify the flag twice
		if opts.EnableDestructiveAPICalls != nil {
			return fmt.Errorf("destructive actions flag already configured: %w", ErrInvalidClientOptions)
		}
		opts.EnableDestructiveAPICalls = gitprovider.BoolVar(destructiveActions)
		return nil
	}
}

// WithConditionalRequests instructs the client to use Conditional Requests to GitHub, asking GitHub
// whether a resource has changed (without burning your quota), and using an in-memory cached "database"
// if so. See: https://developer.github.com/v3/#conditional-requests for more information.
func WithConditionalRequests(conditionalRequests bool) ClientOption {
	return func(opts *clientOptions) error {
		// Make sure the user didn't specify the flag twice
		if opts.EnableConditionalRequests != nil {
			return fmt.Errorf("conditional requests flag already configured: %w", ErrInvalidClientOptions)
		}
		opts.EnableConditionalRequests = gitprovider.BoolVar(conditionalRequests)
		return nil
	}
}

type RoundTripperFactory interface {
	Transport(rt http.RoundTripper) http.RoundTripper
}

// authTransportFactory is a way to acquire a http.RoundTripper with auth credentials configured.
type authTransportFactory interface {
	// Transport returns a http.RoundTripper with auth credentials configured.
	Transport(ctx context.Context) http.RoundTripper
}

// oauth2Auth is an implementation of authTransportFactory.
type oauth2Auth struct {
	token string
}

// Transport returns a http.RoundTripper with auth credentials configured.
func (a *oauth2Auth) Transport(ctx context.Context) http.RoundTripper {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: a.token},
	)
	return oauth2.NewClient(ctx, ts).Transport
}

// patAuth is an implementation of authTransportFactory.
type patAuth struct {
	token string
}

// Transport returns a http.RoundTripper with auth credentials configured.
func (a *patAuth) Transport(ctx context.Context) http.RoundTripper {
	return &github.BasicAuthTransport{
		Username: patUsername,
		Password: a.token,
	}
}

// makeOptions assembles a clientOptions struct from ClientOption mutator functions.
func makeOptions(opts ...ClientOption) (*clientOptions, error) {
	o := &clientOptions{}
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// NewClient creates a new gitprovider.Client instance for GitHub API endpoints.
//
// Using WithOAuth2Token or WithPersonalAccessToken you can specify authentication
// credentials, passing no such ClientOption will allow public read access only.
//
// Basic Auth is not supported because it is deprecated by GitHub, see
// https://developer.github.com/changes/2020-02-14-deprecating-password-auth/
//
// GitHub Enterprise can be used if you specify the domain using the WithDomain option.
//
// You can customize low-level HTTP Transport functionality by using WithRoundTripper.
// You can also use conditional requests (and an in-memory cache) using WithConditionalRequests.
//
// The chain of transports looks like this:
// github.com API <-> Authentication <-> Cache <-> Custom Roundtripper <-> This Client.
func NewClient(ctx context.Context, optFns ...ClientOption) (gitprovider.Client, error) {
	// Complete the options struct
	opts, err := makeOptions(optFns...)
	if err != nil {
		return nil, err
	}

	transport, err := buildTransportChain(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Create a *http.Client using the transport chain
	httpClient := &http.Client{Transport: transport}

	// Create the GitHub client either for the default github.com domain, or
	// a custom enterprise domain if opts.Domain is set to something other than
	// the default.
	var gh *github.Client
	var domain string

	if opts.Domain == nil || *opts.Domain == defaultDomain {
		// No domain or the default github.com used
		domain = defaultDomain
		gh = github.NewClient(httpClient)
	} else {
		// GitHub Enterprise is used
		domain = *opts.Domain
		baseURL := fmt.Sprintf("https://%s/api/v3/", domain)
		uploadURL := fmt.Sprintf("https://%s/api/uploads/", domain)

		if gh, err = github.NewEnterpriseClient(baseURL, uploadURL, httpClient); err != nil {
			return nil, err
		}
	}
	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(gh, domain, destructiveActions), nil
}

// buildTransportChain builds a chain of http.RoundTrippers calling each other as per the
// description in NewClient.
func buildTransportChain(ctx context.Context, opts *clientOptions) (http.RoundTripper, error) {
	// transport will be the http.RoundTripper for the *http.Client given to the Github client.
	var transport http.RoundTripper
	// Get an authenticated http.RoundTripper, if set
	if opts.authTransportFactory != nil {
		transport = opts.authTransportFactory.Transport(ctx)
	}

	// Conditionally enable conditional requests
	if opts.EnableConditionalRequests != nil && *opts.EnableConditionalRequests {
		// Create a new httpcache high-level Transport
		t := httpcache.NewMemoryCacheTransport()
		// Make the httpcache high-level transport use the auth transport "underneath"
		if transport != nil {
			t.Transport = transport
		}
		// Override the transport with our embedded underlying auth transport
		transport = &cacheRoundtripper{t}
	}

	// If a custom roundtripper was set, pipe it through the transport too
	if opts.RoundTripperFactory != nil {
		customTransport := opts.RoundTripperFactory.Transport(transport)
		if customTransport == nil {
			// The lint failure here is a false positive, for some (unknown) reason
			//nolint:goerr113
			return nil, fmt.Errorf("the RoundTripper returned from the RoundTripperFactory must not be nil: %w", ErrInvalidClientOptions)
		}
		transport = customTransport
	}

	return transport, nil
}

type cacheRoundtripper struct {
	t *httpcache.Transport
}

// This function follows the same logic as in github.com/gregjones/httpcache to be able
// to implement our custom roundtripper logic below.
func cacheKey(req *http.Request) string {
	if req.Method == http.MethodGet {
		return req.URL.String()
	}
	return req.Method + " " + req.URL.String()
}

// RoundTrip calls the underlying RoundTrip (using the cache), but invalidates the cache on
// non GET/HEAD requests and non-"200 OK" responses.
func (r *cacheRoundtripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// These two statements are the same as in github.com/gregjones/httpcache Transport.RoundTrip
	// to be able to implement our custom roundtripper below
	cacheKey := cacheKey(req)
	cacheable := (req.Method == "GET" || req.Method == "HEAD") && req.Header.Get("range") == ""

	// If the object isn't a GET or HEAD request, also invalidate the cache of the GET URL
	// as this action will modify the underlying resource (e.g. DELETE/POST/PATCH)
	if !cacheable {
		r.t.Cache.Delete(req.URL.String())
	}
	// Call the underlying roundtrip
	resp, err := r.t.RoundTrip(req)
	// Don't cache anything but "200 OK" requests
	if resp.StatusCode != http.StatusOK {
		r.t.Cache.Delete(cacheKey)
	}
	return resp, err
}
