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
	"fmt"
	"net/url"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"
)

// ProviderID is the provider ID for AzureDevops.
const ProviderID = gitprovider.ProviderID("azureDevops")

func newClient(c core.Client, g git.Client, domain string, destructiveActions bool) *ProviderClient {
	ctx := &clientContext{
		c,
		g,
		domain,
		destructiveActions}
	return &ProviderClient{
		clientContext: ctx,
		orgs: &OrganizationsClient{
			clientContext: ctx,
		},
		repos: &RepositoriesClient{
			clientContext: ctx,
		},
	}
}

// ProviderClient implements the gitprovider.Client interface.
var _ gitprovider.Client = &ProviderClient{}

type clientContext struct {
	c                  core.Client
	g                  git.Client
	domain             string
	destructiveActions bool
}

// ProviderClient is the AzureDevops implementation of the gitprovider.Client interface.
type ProviderClient struct {
	*clientContext

	orgs  *OrganizationsClient
	repos *RepositoriesClient
}

// Raw returns the underlying AzureDevops client.
// It returns the core.Client
func (c *ProviderClient) Raw() interface{} {
	return c.c
}

// UserRepositories returns the UserRepositoriesClient for the client.
func (c *ProviderClient) UserRepositories() gitprovider.UserRepositoriesClient {
	// Method not support for AzureDevops
	return nil
}

// OrgRepositories returns the OrgRepositoriesClient for the client.
func (c *ProviderClient) OrgRepositories() gitprovider.OrgRepositoriesClient {
	return c.repos
}

// Organizations returns the OrganizationsClient for the client.
func (c *ProviderClient) Organizations() gitprovider.OrganizationsClient {
	return c.orgs
}

// SupportedDomain returns the domain endpoint for this client, e.g. "dev.azure.com" or
// "my-custom-git-server.com:6443".
// This field is set at client creation time, and can't be changed.
func (c *ProviderClient) SupportedDomain() string {
	u, _ := url.Parse(c.domain)
	if u.Scheme == "" {
		c.domain = fmt.Sprintf("https://%s", c.domain)
	}
	return c.domain
}

// ProviderID returns the provider ID for this client, e.g. "azuredevops".
func (c *ProviderClient) ProviderID() gitprovider.ProviderID {
	return ProviderID
}

// HasTokenPermission returns whether the token has the given permission.
func (c *ProviderClient) HasTokenPermission(ctx context.Context, permission gitprovider.TokenPermission) (bool, error) {
	return false, gitprovider.ErrNoProviderSupport
}
