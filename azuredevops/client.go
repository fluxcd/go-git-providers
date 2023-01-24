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
	"fmt"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"net/url"
)

// ProviderID is the provider ID for AzureDevops.
const ProviderID = gitprovider.ProviderID("azureDevops")

func newClient(c core.Client, g git.Client, domain string, destructiveActions bool) *Client {
	azureDevopsClient := &azureDevopsClientImpl{c, g, destructiveActions}
	ctx := &clientContext{azureDevopsClient, domain, destructiveActions}
	return &Client{
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
var _ gitprovider.Client = &Client{}

type clientContext struct {
	c                  azureDevopsClient
	domain             string
	destructiveActions bool
}

// Client is an interface that allows talking to a Git provider.
type Client struct {
	*clientContext
	orgs  *OrganizationsClient
	repos *RepositoriesClient
}

func (c *Client) UserRepositories() gitprovider.UserRepositoriesClient {
	//TODO implement me
	panic("implement me")
}

func (c *Client) OrgRepositories() gitprovider.OrgRepositoriesClient {
	return c.repos
}

func (c *Client) Organizations() gitprovider.OrganizationsClient {
	return c.orgs
}

// SupportedDomain returns the domain endpoint for this client, e.g. "gitlab.com" or
// "my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
// what endpoints.
// This field is set at client creation time, and can't be changed.
func (c *Client) SupportedDomain() string {
	u, _ := url.Parse(c.domain)
	if u.Scheme == "" {
		c.domain = fmt.Sprintf("https://%s", c.domain)
	}
	return c.domain
}

func (c *Client) ProviderID() gitprovider.ProviderID {
	return ProviderID
}

func (c Client) HasTokenPermission(ctx context.Context, permission gitprovider.TokenPermission) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (c Client) Raw() interface{} {
	//TODO implement me
	panic("implement me")
}
