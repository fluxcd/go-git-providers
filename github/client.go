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
	"strings"

	"github.com/google/go-github/v66/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// ProviderID is the provider ID for GitHub.
const ProviderID = gitprovider.ProviderID("github")

func newClient(c *github.Client, domain string, destructiveActions bool) *Client {
	ghClient := &githubClientImpl{c, destructiveActions}
	ctx := &clientContext{ghClient, domain, destructiveActions}
	return &Client{
		clientContext: ctx,
		orgs: &OrganizationsClient{
			clientContext: ctx,
		},
		orgRepos: &OrgRepositoriesClient{
			clientContext: ctx,
		},
		userRepos: &UserRepositoriesClient{
			clientContext: ctx,
		},
	}
}

type clientContext struct {
	c                  githubClient
	domain             string
	destructiveActions bool
}

// Client implements the gitprovider.Client interface.
var _ gitprovider.Client = &Client{}

// Client is an interface that allows talking to a Git provider.
type Client struct {
	*clientContext

	orgs      *OrganizationsClient
	orgRepos  *OrgRepositoriesClient
	userRepos *UserRepositoriesClient
}

// SupportedDomain returns the domain endpoint for this client, e.g. "github.com", "enterprise.github.com" or
// "my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
// what endpoints.
// This field is set at client creation time, and can't be changed.
func (c *Client) SupportedDomain() string {
	return c.domain
}

// ProviderID returns the provider ID "github".
// This field is set at client creation time, and can't be changed.
func (c *Client) ProviderID() gitprovider.ProviderID {
	return ProviderID
}

// Raw returns the Go GitHub client (github.com/google/go-github/v47/github *Client)
// used under the hood for accessing GitHub.
func (c *Client) Raw() interface{} {
	return c.c.Client()
}

// Organizations returns the OrganizationsClient handling sets of organizations.
func (c *Client) Organizations() gitprovider.OrganizationsClient {
	return c.orgs
}

// OrgRepositories returns the OrgRepositoriesClient handling sets of repositories in an organization.
func (c *Client) OrgRepositories() gitprovider.OrgRepositoriesClient {
	return c.orgRepos
}

// UserRepositories returns the UserRepositoriesClient handling sets of repositories for a user.
func (c *Client) UserRepositories() gitprovider.UserRepositoriesClient {
	return c.userRepos
}

//nolint:gochecknoglobals
var permissionScopes = map[gitprovider.TokenPermission]string{
	gitprovider.TokenPermissionRWRepository: "repo",
}

// HasTokenPermission returns true if the given token has the given permissions.
func (c *Client) HasTokenPermission(ctx context.Context, permission gitprovider.TokenPermission) (bool, error) {
	requestedScope, ok := permissionScopes[permission]
	if !ok {
		return false, gitprovider.ErrNoProviderSupport
	}

	// The X-OAuth-Scopes header is returned for any API calls, using Meta here to keep things simple.
	_, res, err := c.c.Client().Meta.Get(ctx)
	if err != nil {
		return false, err
	}

	scopes := res.Header.Get("X-OAuth-Scopes")
	if scopes == "" {
		return false, gitprovider.ErrMissingHeader
	}

	for _, s := range strings.Split(scopes, ",") {
		scope := strings.TrimSpace(s)
		if scope == requestedScope {
			return true, nil
		}
	}

	return false, nil
}
