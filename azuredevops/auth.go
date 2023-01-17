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

package azuredevops

import (
	"context"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
)

// NewPatConnection creates a new Client instance for Azure Devops API endpoints.
// The client accepts a domain+token as an argument, which is used to authenticate.
// Variadic parameters gitprovider.ClientOption are used to pass additional options to the gitprovider.Client.

func NewClient(domain string, personalAccessToken string, optFns ...gitprovider.ClientOption) (gitprovider.Client, error) {

	// Complete the options struct
	opts, err := gitprovider.MakeClientOptions(optFns...)
	if err != nil {
		return nil, err
	}
	var azd core.Client
	ctx := context.Background()

	connection := azuredevops.NewPatConnection(domain, personalAccessToken)
	azd, err = core.NewClient(ctx, connection)
	if err != nil {
		return nil, err
	}
	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(azd, domain, destructiveActions), nil
}
