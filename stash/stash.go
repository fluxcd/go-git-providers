/*
Copyright 2021 The Flux authors

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

package stash

import (
	"context"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
	"github.com/go-logr/logr"
)

// ProviderID is the provider ID for BitBucket Server a.k.a Stash.
const (
	// ProviderID is the provider ID for BitBucket Server a.k.a Stash.
	ProviderID = gitprovider.ProviderID("stash")
)

func newClient(c *Client, host, token string, destructiveActions bool, logger logr.Logger) *ProviderClient {
	ctx := &clientContext{
		client:             c,
		host:               host,
		token:              token,
		destructiveActions: destructiveActions,
		log:                logger,
	}

	return &ProviderClient{
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
	client             *Client
	host               string
	token              string
	destructiveActions bool
	log                logr.Logger
}

// Client implements the gitprovider.Client interface.
var _ gitprovider.Client = &ProviderClient{}

// ProviderClient is an interface that allows talking to a Git provider.
type ProviderClient struct {
	*clientContext

	orgs      *OrganizationsClient
	orgRepos  *OrgRepositoriesClient
	userRepos *UserRepositoriesClient
}

// SupportedDomain returns the host endpoint for this client, e.g. "https://mystash.com:7990"
// This allows a higher-level user to know what Client to use for what endpoints.
// This field is set at client creation time, and can't be changed.
func (p *ProviderClient) SupportedDomain() string {
	return p.client.BaseURL.String()
}

// ProviderID returns the provider ID "gostash..
// This field cannot be changed.
func (p *ProviderClient) ProviderID() gitprovider.ProviderID {
	return ProviderID
}

// Raw returns the Go Stash client http.Client
// used under the hood for accessing Stash.
func (p *ProviderClient) Raw() interface{} {
	return p.client.Raw()
}

// Organizations returns the OrganizationsClient handling sets of organizations.
func (p *ProviderClient) Organizations() gitprovider.OrganizationsClient {
	return p.orgs
}

// OrgRepositories returns the OrgRepositoriesClient handling sets of repositories in an organization.
func (p *ProviderClient) OrgRepositories() gitprovider.OrgRepositoriesClient {
	return p.orgRepos
}

// UserRepositories returns the UserRepositoriesClient handling sets of repositories for a user.
func (p *ProviderClient) UserRepositories() gitprovider.UserRepositoriesClient {
	return p.userRepos
}

// HasTokenPermission returns a boolean indicating whether the supplied token has the requested permission.
func (p *ProviderClient) HasTokenPermission(ctx context.Context, permission gitprovider.TokenPermission) (bool, error) {
	return false, gitprovider.ErrNoProviderSupport
}

// validateAPIObject creates a Validatior with the specified name, gives it to fn, and
// depending on if any error was registered with it; either returns nil, or a MultiError
// with both the validation error and ErrInvalidServerData, to mark that the server data
// was invalid.
func validateAPIObject(name string, fn func(validation.Validator)) error {
	v := validation.New(name)
	fn(v)
	// If there was a validation error, also mark it specifically as invalid server data
	if err := v.Error(); err != nil {
		return validation.NewMultiError(err, gitprovider.ErrInvalidServerData)
	}
	return nil
}
