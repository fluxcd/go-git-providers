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

package gitea

import (
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

const (
	// DefaultDomain specifies the default domain used as the backend.
	DefaultDomain = "gitea.com"
	// TokenVariable is the common name for the environment variable
	// containing a Gitea authentication token.
	TokenVariable = "GITEA_TOKEN" // #nosec G101
)

// NewClient creates a new gitprovider.Client instance for Gitea API endpoints.
//
// Gitea Selfhosted can be used if you specify the domain using WithDomain.
func NewClient(optFns ...gitprovider.ClientOption) (gitprovider.Client, error) {
	// Complete the options struct
	opts, err := gitprovider.MakeClientOptions(optFns...)
	if err != nil {
		return nil, err
	}

	// Create a *http.Client using the transport chain
	httpClient, err := gitprovider.BuildClientFromTransportChain(opts.GetTransportChain())
	if err != nil {
		return nil, err
	}

	// Create the Gitea client either for the default gitea.com domain, or
	// a custom enterprise domain if opts.Domain is set to something other than
	// the default.
	var gt *gitea.Client
	var domain string

	// Gitea is primarily self-hosted
	// using test domain if domain not provided
	domain = *opts.Domain
	if opts.Domain == nil || *opts.Domain == DefaultDomain {
		// No domain set or the default gitea.com used
		domain = DefaultDomain
	}
	baseURL := domain
	if !strings.Contains(domain, "://") {
		baseURL = fmt.Sprintf("https://%s/", domain)
	}
	if gt, err = gitea.NewClient(baseURL, gitea.SetHTTPClient(httpClient)); err != nil {
		return nil, err
	}
	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(gt, domain, destructiveActions), nil
}
