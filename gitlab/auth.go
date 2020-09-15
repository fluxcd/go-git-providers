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

package gitlab

import (
	"fmt"

	"github.com/fluxcd/go-git-providers/gitprovider"
	gogitlab "github.com/xanzy/go-gitlab"
)

const (
	// DefaultDomain specifies the default domain used as the backend.
	DefaultDomain = "gitlab.com"
)

// ClientOption is the interface to implement for passing options to NewClient.
// The clientOptions struct is private to force usage of the With... functions.
type ClientOption interface {
	// ApplyToGitlabClientOptions applies set fields of this object into target.
	ApplyToGitlabClientOptions(target *clientOptions) error
}

// clientOptions is the struct that tracks data about what options have been set.
type clientOptions struct {
	// clientOptions shares all the common options
	gitprovider.CommonClientOptions

	// AuthTransport is a ChainableRoundTripperFunc adding authentication credentials to the transport chain.
	AuthTransport gitprovider.ChainableRoundTripperFunc
}

// ApplyToGitlabClientOptions implements ClientOption, and applies the set fields of opts
// into target. If both opts and target has the same specific field set, ErrInvalidClientOptions is returned.
func (opts *clientOptions) ApplyToGitlabClientOptions(target *clientOptions) error {
	// Apply common values, if any
	if err := opts.CommonClientOptions.ApplyToCommonClientOptions(&target.CommonClientOptions); err != nil {
		return err
	}

	return nil
}

// getTransportChain builds the full chain of transports (from left to right,
// as per gitprovider.BuildClientFromTransportChain) of the form described in NewClient.
func (opts *clientOptions) getTransportChain() (chain []gitprovider.ChainableRoundTripperFunc) {
	if opts.PostChainTransportHook != nil {
		chain = append(chain, opts.PostChainTransportHook)
	}
	if opts.AuthTransport != nil {
		chain = append(chain, opts.AuthTransport)
	}
	if opts.PreChainTransportHook != nil {
		chain = append(chain, opts.PreChainTransportHook)
	}
	return
}

// buildCommonOption is a helper for returning a ClientOption out of a common option field.
func buildCommonOption(opt gitprovider.CommonClientOptions) *clientOptions {
	return &clientOptions{CommonClientOptions: opt}
}

// errorOption implements ClientOption, and just wraps an error which is immediately returned.
// This struct can be used through the optionError function, in order to make makeOptions fail
// if there are invalid options given to the With... functions.
type errorOption struct {
	err error
}

// ApplyToGitlabClientOptions implements ClientOption, but just returns the internal error.
func (e *errorOption) ApplyToGitlabClientOptions(*clientOptions) error { return e.err }

// optionError is a constructor for errorOption.
func optionError(err error) ClientOption {
	return &errorOption{err}
}

//
// Common options
//

// WithDomain initializes a Client for a custom GitLab instance of the given domain.
// Only host and port information should be present in domain. domain must not be an empty string.
func WithDomain(domain string) ClientOption {
	return buildCommonOption(gitprovider.CommonClientOptions{Domain: &domain})
}

// WithDestructiveAPICalls tells the client whether it's allowed to do dangerous and possibly destructive
// actions, like e.g. deleting a repository.
func WithDestructiveAPICalls(destructiveActions bool) ClientOption {
	return buildCommonOption(gitprovider.CommonClientOptions{EnableDestructiveAPICalls: &destructiveActions})
}

// WithPreChainTransportHook registers a ChainableRoundTripperFunc "before" the cache and authentication
// transports in the chain. For more information, see NewClient, and gitprovider.CommonClientOptions.PreChainTransportHook.
func WithPreChainTransportHook(preRoundTripperFunc gitprovider.ChainableRoundTripperFunc) ClientOption {
	// Don't allow an empty value
	if preRoundTripperFunc == nil {
		return optionError(fmt.Errorf("preRoundTripperFunc cannot be nil: %w", gitprovider.ErrInvalidClientOptions))
	}

	return buildCommonOption(gitprovider.CommonClientOptions{PreChainTransportHook: preRoundTripperFunc})
}

// WithPostChainTransportHook registers a ChainableRoundTripperFunc "after" the cache and authentication
// transports in the chain. For more information, see NewClient, and gitprovider.CommonClientOptions.WithPostChainTransportHook.
func WithPostChainTransportHook(postRoundTripperFunc gitprovider.ChainableRoundTripperFunc) ClientOption {
	// Don't allow an empty value
	if postRoundTripperFunc == nil {
		return optionError(fmt.Errorf("postRoundTripperFunc cannot be nil: %w", gitprovider.ErrInvalidClientOptions))
	}

	return buildCommonOption(gitprovider.CommonClientOptions{PostChainTransportHook: postRoundTripperFunc})
}

// makeOptions assembles a clientOptions struct from ClientOption mutator functions.
func makeOptions(opts ...ClientOption) (*clientOptions, error) {
	o := &clientOptions{}
	for _, opt := range opts {
		if err := opt.ApplyToGitlabClientOptions(o); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// NewClientFromPAT creates a new gitlab.Client instance for GitLab API endpoints.
func NewClientFromPAT(personalAccessToken string, optFns ...ClientOption) (gitprovider.Client, error) {
	var gl *gogitlab.Client
	var domain, sshDomain string

	// Complete the options struct
	opts, err := makeOptions(optFns...)
	if err != nil {
		return nil, err
	}

	// Create a *http.Client using the transport chain
	httpClient, err := gitprovider.BuildClientFromTransportChain(opts.getTransportChain())
	if err != nil {
		return nil, err
	}

	if opts.Domain == nil || *opts.Domain == DefaultDomain {
		// No domain set or the default gitlab.com used
		domain = DefaultDomain
		gl, err = gogitlab.NewClient(personalAccessToken, gogitlab.WithHTTPClient(httpClient))
		if err != nil {
			return nil, err
		}
	}

	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}
	fmt.Println("client at this point: ", gl)
	return newClient(gl, domain, sshDomain, destructiveActions), nil
}

// NewClientFromUsernamePassword creates a new gitlab.Client instance for GitLab API endpoints.
func NewClientFromUsernamePassword(username string, password string, optFns ...ClientOption) (gitprovider.Client, error) {
	var gl *gogitlab.Client
	var domain, sshDomain string

	// Complete the options struct
	opts, err := makeOptions(optFns...)
	if err != nil {
		return nil, err
	}

	// Create a *http.Client using the transport chain
	httpClient, err := gitprovider.BuildClientFromTransportChain(opts.getTransportChain())
	if err != nil {
		return nil, err
	}

	if opts.Domain == nil || *opts.Domain == DefaultDomain {
		// No domain set or the default gitlab.com used
		domain = DefaultDomain
		gl, err = gogitlab.NewBasicAuthClient(username, password, gogitlab.WithHTTPClient(httpClient))
		if err != nil {
			return nil, err
		}
	}

	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(gl, domain, sshDomain, destructiveActions), nil
}

// NewClientFromOAuthToken creates a new gitlab.Client instance for GitLab API endpoints.
func NewClientFromOAuthToken(oauthAccessToken string, optFns ...ClientOption) (gitprovider.Client, error) {
	var gl *gogitlab.Client
	var domain, sshDomain string

	// Complete the options struct
	opts, err := makeOptions(optFns...)
	if err != nil {
		return nil, err
	}

	// Create a *http.Client using the transport chain
	httpClient, err := gitprovider.BuildClientFromTransportChain(opts.getTransportChain())
	if err != nil {
		return nil, err
	}

	if opts.Domain == nil || *opts.Domain == DefaultDomain {
		// No domain set or the default gitlab.com used
		domain = DefaultDomain
		gl, err = gogitlab.NewOAuthClient(oauthAccessToken, gogitlab.WithHTTPClient(httpClient))
		if err != nil {
			return nil, err
		}
	}

	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(gl, domain, sshDomain, destructiveActions), nil
}
