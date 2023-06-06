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

package gitea

import (
	"context"

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface.
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on organizations the user has access to.
type OrganizationsClient struct {
	*clientContext
}

// Get a specific organization the user has access to.
// This can't refer to a sub-organization in Gitea, as those aren't supported.
//
// ErrNotFound is returned if the resource does not exist.
func (c *OrganizationsClient) Get(ctx context.Context, ref gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	// Make sure the OrganizationRef is valid
	if err := validateOrganizationRef(ref, c.domain); err != nil {
		return nil, err
	}

	// GET /orgs/{org}
	apiObj, err := c.getOrg(ref.Organization)
	if err != nil {
		return nil, err
	}

	return newOrganization(c.clientContext, apiObj, ref), nil
}

// List all top-level organizations the specific user has access to.
//
// List returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	// GET /user/orgs
	apiObjs, err := c.listOrgs()
	if err != nil {
		return nil, err
	}

	orgs := make([]gitprovider.Organization, 0, len(apiObjs))
	for _, apiObj := range apiObjs {
		// apiObj.Login is already validated to be non-nil in ListOrgs
		orgs = append(orgs, newOrganization(c.clientContext, apiObj, gitprovider.OrganizationRef{
			Domain:       c.domain,
			Organization: apiObj.UserName,
		}))
	}

	return orgs, nil
}

// getOrg returns a specific organization the user has access to.
func (c *OrganizationsClient) getOrg(orgName string) (*gitea.Organization, error) {
	apiObj, res, err := c.c.GetOrg(orgName)
	if err != nil {
		return nil, handleHTTPError(res, err)
	}
	// Validate the API object
	if err := validateOrganizationAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

// listOrgs returns all of current user's organizations.
func (c *OrganizationsClient) listOrgs() ([]*gitea.Organization, error) {
	opts := gitea.ListOrgsOptions{}
	apiObjs := []*gitea.Organization{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /user/orgs"
		pageObjs, resp, listErr := c.c.ListMyOrgs(opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateOrganizationAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

// Children returns the immediate child-organizations for the specific OrganizationRef o.
// The OrganizationRef may point to any existing sub-organization.
//
// This is not supported in Gitea.
//
// Children returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) Children(_ context.Context, _ gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}
