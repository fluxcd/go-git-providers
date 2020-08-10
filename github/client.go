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

package github

import (
	"context"

	gitprovider "github.com/fluxcd/go-git-providers"
	"github.com/google/go-github/v32/github"
)

// ProviderID is the provider ID for GitHub
const ProviderID = gitprovider.ProviderID("github")

func newClient(c *github.Client, domain string) *Client {
	ctx := &clientContext{c, domain}
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

type clientContext struct {
	c      *github.Client
	domain string
}

// Client implements the gitprovider.Client interface
var _ gitprovider.Client = &Client{}

type Client struct {
	*clientContext

	orgs  *OrganizationsClient
	repos *RepositoriesClient
}

// SupportedDomain returns the domain endpoint for this client, e.g. "github.com", "enterprise.github.com" or
// "my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
// what endpoints.
// This field is set at client creation time, and can't be changed
func (c *Client) SupportedDomain() string {
	return c.domain
}

// ProviderID returns the provider ID "github"
// This field is set at client creation time, and can't be changed
func (c *Client) ProviderID() gitprovider.ProviderID {
	return ProviderID
}

// Raw returns the Go GitHub client (github.com/google/go-github/v32/github *Client)
// used under the hood for accessing GitHub.
func (c *Client) Raw() interface{} {
	return c.c
}

// Organization gets the OrganizationClient for a specific top-level organization.
// It is ensured that the organization the reference points to exists, as it's looked up
// and returned as the second argument.
//
// ErrNotTopLevelOrganization will be returned at usage time if the organization is not top-level.
// ErrNotFound is returned if the organization does not exist.
func (c *Client) Organization(ctx context.Context, o gitprovider.OrganizationRef) (gitprovider.OrganizationClient, *gitprovider.Organization, error) {
	org, err := c.orgs.Get(ctx, o)
	if err != nil {
		return nil, nil, err
	}
	return newOrganizationClient(c.clientContext, org), org, nil
}

// Organizations returns the OrganizationsClient handling sets of organizations.
func (c *Client) Organizations() gitprovider.OrganizationsClient {
	return c.orgs
}

// Repository gets the RepositoryClient for the specified RepositoryRef.
// It is ensured that the repository the reference points to exists, as it's looked up
// and returned as the second argument.
//
// ErrNotFound is returned if the repository does not exist.
func (c *Client) Repository(ctx context.Context, r gitprovider.RepositoryRef) (gitprovider.RepositoryClient, *gitprovider.Repository, error) {
	repo, err := c.repos.Get(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	return newRepositoryClient(c.clientContext, repo), repo, nil
}

// Repositories returns the RepositoriesClient handling sets of organizations.
func (c *Client) Repositories() gitprovider.RepositoriesClient {
	return c.repos
}
