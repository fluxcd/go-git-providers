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
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface.
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on the groups the user has access to.
type OrganizationsClient struct {
	*clientContext
}

// Get a specific group the user has access to.
// This can refer to a sub-group in GitLab.
//
// ErrNotFound is returned if the resource does not exist.
func (c *OrganizationsClient) Get(ctx context.Context, ref gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	// GET /groups/{group}
	apiObj, err := c.c.GetGroup(ctx, ref.Organization)
	if err != nil {
		return nil, err
	}

	return newOrganization(c.clientContext, apiObj, ref), nil
}

// List all groups the specific user has access to.
//
// List returns all available groups, using multiple paginated requests if needed.
func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	// GET /groups
	apiObjs, err := c.c.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	groups := make([]gitprovider.Organization, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		groups = append(groups, newOrganization(c.clientContext, apiObj, gitprovider.OrganizationRef{}))
	}

	return groups, nil
}

// Children returns the immediate child-organizations for the specific OrganizationRef o.
// The OrganizationRef may point to any existing sub-organization.
//
// Children returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) Children(_ context.Context, _ gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}
