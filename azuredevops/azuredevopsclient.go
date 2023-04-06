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
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/git"
	"net/url"
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

// Client implements the gitprovider.Client interface.
var _ gitprovider.Client = &ProviderClient{}

type clientContext struct {
	c                  core.Client
	g                  git.Client
	domain             string
	destructiveActions bool
}

type ProviderClient struct {
	*clientContext

	orgs  *OrganizationsClient
	repos *RepositoriesClient
}

func (c *ProviderClient) Raw() interface{} {
	return gitprovider.ErrNoProviderSupport
}

func (c *ProviderClient) UserRepositories() gitprovider.UserRepositoriesClient {
	// Method not support for AzureDevops
	return nil
}

func (c *ProviderClient) OrgRepositories() gitprovider.OrgRepositoriesClient {
	return c.repos
}

func (c *ProviderClient) Organizations() gitprovider.OrganizationsClient {
	return c.orgs
}

// SupportedDomain returns the domain endpoint for this client, e.g. "gitlab.com" or
// "my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
// what endpoints.
// This field is set at client creation time, and can't be changed.
func (c *ProviderClient) SupportedDomain() string {
	u, _ := url.Parse(c.domain)
	if u.Scheme == "" {
		c.domain = fmt.Sprintf("https://%s", c.domain)
	}
	return c.domain
}

func (c *ProviderClient) ProviderID() gitprovider.ProviderID {
	return ProviderID
}

func (c *ProviderClient) HasTokenPermission(ctx context.Context, permission gitprovider.TokenPermission) (bool, error) {
	return false, gitprovider.ErrNoProviderSupport
}
