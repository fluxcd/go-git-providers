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

package gitlab

import (
	"context"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/xanzy/go-gitlab"
)

// ProviderID is the provider ID for GitLab.
const ProviderID = gitprovider.ProviderID("gitlab")

func newClient(c *gitlab.Client, domain string, sshDomain string, destructiveActions bool) *Client {
	glClient := &gitlabClientImpl{c, destructiveActions}
	ctx := &clientContext{glClient, domain, sshDomain, destructiveActions}
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
	c                  gitlabClient
	domain             string
	sshDomain          string
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

// SupportedDomain returns the domain endpoint for this client, e.g. "gitlab.com" or
// "my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
// what endpoints.
// This field is set at client creation time, and can't be changed.
func (c *Client) SupportedDomain() string {
	return c.domain
}

// SupportedSSHDomain returns the ssh domain endpoint for this client, e.g. "gitlab.com" or
// "ssh.my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
// what endpoints.
// This field is set at client creation time, and can't be changed.
func (c *Client) SupportedSSHDomain() string {
	return c.sshDomain
}

// ProviderID returns the provider ID "gitlab".
// This field is set at client creation time, and can't be changed.
func (c *Client) ProviderID() gitprovider.ProviderID {
	return ProviderID
}

// Raw returns the Go GitLab client (github.com/xanzy *Client)
// used under the hood for accessing GitLab.
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

func (c *Client) HasTokenPermission(ctx context.Context, permission gitprovider.TokenPermission) (bool, error) {
	return false, gitprovider.ErrNoProviderSupport
}
