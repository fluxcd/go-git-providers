/*
Copyright 2023 The Flux CD contributors.

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
	"errors"
	"fmt"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"

	"net/url"
)

// NewClient creates a new Client instance for Azure Devops API endpoints.
// The client accepts a personal token used for which is used to authenticate and context as an argument,
// Variadic parameters gitprovider.ClientOption are used to pass additional options to the gitprovider.Client.

func NewClient(personalAccessToken string, ctx context.Context, optFns ...gitprovider.ClientOption) (gitprovider.Client, error) {

	// Complete the options struct
	opts, err := gitprovider.MakeClientOptions(optFns...)
	if err != nil {
		return nil, err
	}

	if opts.Domain == nil {
		return nil, errors.New("please provide the domain url with the project path ")
	}
	// This is the link to the project/organization
	domain := *opts.Domain
	u, err := url.Parse(domain)
	if err != nil {
		return nil, fmt.Errorf(" URL parsing failed: %v", err)
	}
	if u.Scheme == "" || u.Scheme == "http" {
		domain = fmt.Sprintf("https://%s%s", u.Host, u.Path)
	}
	// azuredevops.NewPatConnection uses the project and personalAccessToken to connect to Azure
	connection := azuredevops.NewPatConnection(domain, personalAccessToken)
	// coreClient provides access to Azure Devops organization,projects and teams
	coreClient, err := core.NewClient(ctx, connection)
	// gitClient provides access to the Azure Devops Git repositories the files,trees,commits and refs
	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		return nil, err
	}
	// By default, turn destructive actions off. But allow overrides.
	destructiveActions := false
	if opts.EnableDestructiveAPICalls != nil {
		destructiveActions = *opts.EnableDestructiveAPICalls
	}

	return newClient(coreClient, gitClient, domain, destructiveActions), nil
}
