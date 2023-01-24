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
	"errors"
	"fmt"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"net/url"
)

// NewPatConnection creates a new Client instance for Azure Devops API endpoints.
// The client accepts a domain+token as an argument, which is used to authenticate.
// Variadic parameters gitprovider.ClientOption are used to pass additional options to the gitprovider.Client.

func NewClient(personalAccessToken string, optFns ...gitprovider.ClientOption) (gitprovider.Client, error) {
	var connection *azuredevops.Connection
	// coreClient provides access to https://learn.microsoft.com/en-us/rest/api/azure/devops/core which has details about the projects
	var coreClient core.Client
	// gitClient provides access to https://learn.microsoft.com/en-us/rest/api/azure/devops/git which has details about the git repository
	var gitClient git.Client
	var domain string
	// Complete the options struct
	opts, err := gitprovider.MakeClientOptions(optFns...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	if opts.Domain == nil {
		return nil, errors.New("please provide the domain url with the project path ")
	}

	domain = *opts.Domain
	u, _ := url.Parse(domain)
	if u.Scheme == "" || u.Scheme == "http" {
		domain = fmt.Sprintf("https://%s%s", u.Host, u.Path)
	}

	connection = azuredevops.NewPatConnection(domain, personalAccessToken)

	coreClient, err = core.NewClient(ctx, connection)
	gitClient, err = git.NewClient(ctx, connection)
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
