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
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v45/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
	dbg "github.com/gmlewis/go-httpdebug/httpdebug"
)

const (
	// DefaultDomain specifies the default domain used as the backend.
	DefaultDomain = "github.com"
	// TokenVariable is the common name for the environment variable
	// containing a GitHub authentication token.
	TokenVariable = "GITHUB_TOKEN" // #nosec G101
)

// NewClient creates a new gitprovider.Client instance for GitHub API endpoints.
//
// Using WithOAuth2Token you can specify authentication
// credentials, passing no such ClientOption will allow public read access only.
//
// Password-based authentication is not supported because it is deprecated by GitHub, see
// https://developer.github.com/changes/2020-02-14-deprecating-password-auth/
//
// GitHub Enterprise can be used if you specify the domain using WithDomain.
//
// You can customize low-level HTTP Transport functionality by using the With{Pre,Post}ChainTransportHook options.
// You can also use conditional requests (and an in-memory cache) using WithConditionalRequests.
//
// The chain of transports looks like this:
// github.com API <-> "Post Chain" <-> Authentication <-> Cache <-> "Pre Chain" <-> *github.Client.
func NewClient(optFns ...gitprovider.ClientOption) (gitprovider.Client, error) {
	// Complete the options struct
	opts, err := gitprovider.MakeClientOptions(optFns...)
	if err != nil {
		return nil, err
	}

	if opts.Logger == nil {
		logger := logr.Discard()
		opts.Logger = &logger
	}

	// Create a *http.Client using the transport chain
	httpClient, err := gitprovider.BuildClientFromTransportChain(opts.GetTransportChain())
	if err != nil {
		return nil, err
	}

	// add debug
	ct := dbg.New(dbg.WithTransport(httpClient.Transport))
	httpClient = ct.Client()

	// Create the GitHub client either for the default github.com domain, or
	// a custom enterprise domain if opts.Domain is set to something other than
	// the default.
	var gh *github.Client
	var domain string

	if opts.Domain == nil || *opts.Domain == DefaultDomain {
		// No domain or the default github.com used
		domain = DefaultDomain
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

	return newClient(gh, domain, destructiveActions, opts.Logger), nil
}
