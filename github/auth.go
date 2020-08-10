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

	gitprovider "github.com/fluxcd/go-git-providers"
	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

const (
	// defaultDomain specifies the default domain used as the backend
	defaultDomain = "github.com"
	// patUsername is the "username" for the basic auth authentication flow
	// when using a personal access token as the "password". This string could
	// be arbitrary, even unset, as it is not respected server-side. For conventions'
	// sake, we'll set this to "git".
	patUsername = "git"
)

var (
	// ErrInvalidClientOption is the error returned when calling NewClient() with
	// invalid options (e.g. specifying mutually exclusive options)
	ErrInvalidClientOption = errors.New("invalid options given to NewClient()")
)

// clientOptions is the struct that tracks data about what options have been set
// It is private so that the user must use the With... functions
type clientOptions struct {
	// Domain specifies the backing domain, which can be arbitrary if the user uses
	// Github Enterprise. If unset, defaultDomain will be used.
	Domain *string

	// ClientFactory is a way to aquire a *http.Client, possibly with auth credentials
	ClientFactory ClientFactory
}

// ClientOption is a function that is mutating a pointer to a clientOptions object
// which holds information of how the Client should be initialized.
type ClientOption func(*clientOptions) error

// WithOAuth2Token initializes a Client which authenticates with Github through an OAuth2 token.
// oauth2Token must not be an empty string.
// WithOAuth2Token is mutually exclusive with WithPersonalAccessToken and WithClientFactory.
func WithOAuth2Token(oauth2Token string) ClientOption {
	return func(opts *clientOptions) error {
		// Don't allow an empty value
		if len(oauth2Token) == 0 {
			return fmt.Errorf("oauth2Token cannot be empty: %w", ErrInvalidClientOption)
		}
		// Make sure the user didn't specify auth twice
		if opts.ClientFactory != nil {
			return fmt.Errorf("http.Client already configured: %w", ErrInvalidClientOption)
		}
		opts.ClientFactory = &oauth2Auth{oauth2Token}
		return nil
	}
}

// WithPersonalAccessToken initializes a Client which authenticates with Github through a personal access token.
// patToken must not be an empty string.
// WithPersonalAccessToken is mutually exclusive with WithOAuth2Token and WithClientFactory.
func WithPersonalAccessToken(patToken string) ClientOption {
	return func(opts *clientOptions) error {
		// Don't allow an empty value
		if len(patToken) == 0 {
			return fmt.Errorf("patToken cannot be empty: %w", ErrInvalidClientOption)
		}
		// Make sure the user didn't specify auth twice
		if opts.ClientFactory != nil {
			return fmt.Errorf("http.Client already configured: %w", ErrInvalidClientOption)
		}
		opts.ClientFactory = &oauth2Auth{patToken}
		return nil
	}
}

// WithClientFactory initializes a Client with a given ClientFactory, used for aquiring the *http.Client later.
// clientFactory must not be nil.
// WithClientFactory is mutually exclusive with WithOAuth2Token and WithPersonalAccessToken.
func WithClientFactory(clientFactory ClientFactory) ClientOption {
	return func(opts *clientOptions) error {
		// Don't allow an empty value
		if clientFactory == nil {
			return fmt.Errorf("clientFactory cannot be nil: %w", ErrInvalidClientOption)
		}
		// Make sure the user didn't specify auth twice
		if opts.ClientFactory != nil {
			return fmt.Errorf("http.Client already configured: %w", ErrInvalidClientOption)
		}
		opts.ClientFactory = clientFactory
		return nil
	}
}

// WithDomain initializes a Client for a custom GitHub Enterprise instance of the given domain.
// Only host and port information should be present in domain. domain must not be an empty string.
func WithDomain(domain string) ClientOption {
	return func(opts *clientOptions) error {
		// Don't set an empty value
		if len(domain) == 0 {
			return fmt.Errorf("domain cannot be empty: %w", ErrInvalidClientOption)
		}
		// Make sure the user didn't specify the domain twice
		if opts.Domain != nil {
			return fmt.Errorf("domain already configured: %w", ErrInvalidClientOption)
		}
		opts.Domain = gitprovider.StringVar(domain)
		return nil
	}
}

// ClientFactory is a way to aquire a *http.Client, possibly with auth credentials
type ClientFactory interface {
	// Client returns a *http.Client, possibly with auth credentials
	Client(ctx context.Context) *http.Client
}

// oauth2Auth is an implementation of ClientFactory
type oauth2Auth struct {
	token string
}

// Client returns a *http.Client, possibly with auth credentials
func (a *oauth2Auth) Client(ctx context.Context) *http.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: a.token},
	)
	return oauth2.NewClient(ctx, ts)
}

// patAuth is an implementation of ClientFactory
type patAuth struct {
	token string
}

// Client returns a *http.Client, possibly with auth credentials
func (a *patAuth) Client(ctx context.Context) *http.Client {
	auth := github.BasicAuthTransport{
		Username: patUsername,
		Password: a.token,
	}
	return auth.Client()
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

// NewClient creates a new gitprovider.Client instance for Github API endpoints.
//
// Using WithOAuth2Token or WithPersonalAccessToken you can specify authentication
// credentials, given no such ClientOption will allow public read access only.
//
// Basic Auth is not supported because it is deprecated by Github, see
// https://developer.github.com/changes/2020-02-14-deprecating-password-auth/
//
// GitHub Enterprise can be used if you specify the domain using the WithDomain option.
func NewClient(ctx context.Context, optFns ...ClientOption) (gitprovider.Client, error) {
	// Complete the options struct
	opts, err := makeOptions(optFns...)
	if err != nil {
		return nil, err
	}

	// Get the *http.Client to use for the transport, possibly with authentication.
	// If opts.ClientFactory is nil, just leave the httpClient as nil, it will be
	// automatically set by the github package.
	var httpClient *http.Client
	if opts.ClientFactory != nil {
		httpClient = opts.ClientFactory.Client(ctx)
	}

	// Create the Github client either for the default github.com domain, or
	// a custom enterprise domain if opts.Domain is set to something other than
	// the default.
	var gh *github.Client
	var domain string
	if opts.Domain == nil || *opts.Domain == defaultDomain {
		// No domain or the default github.com used
		domain = defaultDomain
		gh = github.NewClient(httpClient)
	} else {
		// Github Enterprise is used
		domain = *opts.Domain
		baseURL := fmt.Sprintf("https://%s/api/v3/", domain)
		uploadURL := fmt.Sprintf("https://%s/api/uploads/", domain)

		if gh, err = github.NewEnterpriseClient(baseURL, uploadURL, httpClient); err != nil {
			return nil, err
		}
	}

	return newClient(gh, domain), nil
}
