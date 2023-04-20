/*
Copyright 2021 The Flux authors

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
	"errors"
	"fmt"
	"net/url"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/go-logr/logr"
)

// NewStashClient creates a new Client instance for Stash API endpoints.
// The client accepts a username+token as an argument, which is used to authenticate.
// The host name is used to construct the base URL for the Stash API.
// Variadic parameters gitprovider.ClientOption are used to pass additional options to the gitprovider.Client.
func NewStashClient(username, token string, optFns ...gitprovider.ClientOption) (*ProviderClient, error) {
	url := &url.URL{}

	opts, err := gitprovider.MakeClientOptions(optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed making client options: %w", err)
	}

	// Create a *http.Client using the transport chain
	client, err := gitprovider.BuildClientFromTransportChain(opts.GetTransportChain())
	if err != nil {
		return nil, fmt.Errorf("failed building client: %w", err)
	}

	if opts.Domain == nil {
		return nil, errors.New("host is required")
	}

	host := *opts.Domain

	logger := logr.Discard()
	if opts.Logger != nil {
		logger = *opts.Logger
	}

	url, err = url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("failed parsing host URL %q: %w", host, err)
	}

	var stashClient *Client
	if len(opts.CABundle) != 0 {
		stashClient, err = NewClient(client, host, nil, logger, WithAuth(username, token), WithCABundle(opts.CABundle))
	} else {
		stashClient, err = NewClient(client, host, nil, logger, WithAuth(username, token))
	}

	if err != nil {
		return nil, fmt.Errorf("failed creating client: %w", err)
	}

	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(stashClient, host, token, destructiveActions, logger), nil
}
