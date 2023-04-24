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
	"strconv"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface.
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on the groups the user has access to.
type OrganizationsClient struct {
	*clientContext
}

// Get a specific project that a user has access to
func (c *OrganizationsClient) Get(ctx context.Context, ref gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	opts := core.GetProjectArgs{ProjectId: &ref.Organization}
	project, err := c.c.GetProject(ctx, opts)
	if err != nil {
		return nil, err
	}

	ref.SetKey(project.Id.String())
	return newProject(c.clientContext, *project, ref), nil
}

// List all the projects the specific user has access to.
// List returns all available projects, using multiple paginated requests if needed.
func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	opts := core.GetProjectsArgs{}
	apiObjs, err := c.c.GetProjects(ctx, opts)
	if err != nil {
		return nil, err
	}

	projects := make([]gitprovider.Organization, len(apiObjs.Value))

	index := 0
	for apiObjs != nil {
		for i, apiObj := range apiObjs.Value {
			ref := gitprovider.OrganizationRef{
				Domain:       *apiObj.Url,
				Organization: *apiObj.Name,
			}

			teamProject := core.TeamProject{
				Id:   apiObj.Id,
				Name: apiObj.Name,
			}

			ref.SetKey(apiObj.Id.String())
			projects[i] = newProject(c.clientContext, teamProject, ref)
			index++
		}
		if apiObjs.ContinuationToken != "" {
			continuationToken, err := strconv.Atoi(apiObjs.ContinuationToken)
			if err != nil {
				return nil, fmt.Errorf("Error converting 'ContinuationToken' to integer: %w", err)
			}

			opts := core.GetProjectsArgs{
				ContinuationToken: &continuationToken,
			}
			apiObjs, err = c.c.GetProjects(ctx, opts)
			if err != nil {
				return nil, err
			}
		} else {
			apiObjs = nil
		}
	}

	return projects, nil

}

// Children returns the immediate child-organizations for the specific OrganizationRef o.
// The OrganizationRef may point to any existing sub-organization.
// Children returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) Children(_ context.Context, _ gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}
