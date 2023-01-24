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
	//TODO implement me
	panic("implement me")
}

func (c *ProviderClient) UserRepositories() gitprovider.UserRepositoriesClient {
	//TODO implement me
	panic("implement me")
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
	//TODO implement me
	panic("implement me")
}

func (c *azureDevopsClientImpl) GetProject(ctx context.Context, projectName *string) (*core.TeamProject, error) {
	opts := core.GetProjectArgs{ProjectId: projectName}
	apiObj, err := c.c.GetProject(ctx, opts)
	return apiObj, err
}

func (c *azureDevopsClientImpl) ListPullRequests(ctx context.Context, repositoryId *string) ([]git.GitPullRequest, error) {
	apiObj, err := c.g.GetPullRequests(ctx, git.GetPullRequestsArgs{RepositoryId: repositoryId})
	if err != nil {
		return nil, err
	}
	return *apiObj, nil
}
