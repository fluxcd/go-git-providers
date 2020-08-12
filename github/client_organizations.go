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
	"fmt"

	"github.com/google/go-github/v32/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on organizations the user has access to.
type OrganizationsClient struct {
	*clientContext
}

// Get a specific organization the user has access to.
// This can't refer to a sub-organization in GitHub, as those aren't supported.
//
// ErrNotFound is returned if the resource does not exist.
func (c *OrganizationsClient) Get(ctx context.Context, ref gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	// Make sure the OrganizationRef is valid
	if err := validateOrganizationRef(ref, c.domain); err != nil {
		return nil, err
	}

	// GET /orgs/{org}
	apiObj, _, err := c.c.Organizations.Get(ctx, ref.Organization)
	if err != nil {
		return nil, handleHTTPError(err)
	}

	return newOrganization(c.clientContext, apiObj, ref), nil
}

// List all top-level organizations the specific user has access to.
//
// List returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	apiObjs := []*github.Organization{}
	opts := &github.ListOptions{}
	err := allPages(opts, func() (*github.Response, error) {
		// GET /user/orgs
		pageObjs, resp, listErr := c.c.Organizations.List(ctx, "", opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}

	orgs := make([]gitprovider.Organization, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// Make sure name isn't nil
		if apiObj.Login == nil {
			return nil, fmt.Errorf("didn't expect login to be nil for org: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
		}
		orgs = append(orgs, newOrganization(c.clientContext, apiObj, gitprovider.OrganizationRef{
			Domain:       c.domain,
			Organization: *apiObj.Login,
		}))
	}

	return orgs, nil
}

// Children returns the immediate child-organizations for the specific OrganizationRef o.
// The OrganizationRef may point to any existing sub-organization.
//
// This is not supported in GitHub.
//
// Children returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) Children(_ context.Context, _ gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}
