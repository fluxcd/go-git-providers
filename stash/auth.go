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
	"errors"
	"net/http"
	"net/url"

	"github.com/fluxcd/go-git-providers/gitprovider"
	gostash "github.com/fluxcd/go-git-providers/go-stash"
	"github.com/go-logr/logr"
)

// NewClient creates a new Client instance for Stash API endpoints.
// The client accepts a token as an argument, which is used to authenticate.
// The domain name is used to construct the base URL for the Stash API.
// Variadic parameters gitprovider.ClientOption are used to pass additional options to the gitprovider.Client.
func NewClient(token string, optFns ...gitprovider.ClientOption) (*Client, error) {
	var domain string
	var url = &url.URL{}
	var logger logr.Logger

	opts, err := gitprovider.MakeClientOptions(optFns...)
	if err != nil {
		return nil, err
	}

	// Create a *http.Client using the transport chain
	client, err := gitprovider.BuildClientFromTransportChain(opts.GetTransportChain())
	if err != nil {
		return nil, err
	}

	if opts.Domain == nil {
		return nil, errors.New("domain is required")
	} else {
		domain = *opts.Domain
	}

	if opts.Logger != nil {
		logger = *opts.Logger
	} else {
		logger = logr.Discard()
	}

	url, err = url.Parse(domain)
	if err != nil {
		return nil, err
	}

	// set the auth token	header
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	st, err := gostash.NewClient(client, domain, &header, &logger)
	if err != nil {
		return nil, err
	}

	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(st, domain, token, destructiveActions, logger), nil
}
