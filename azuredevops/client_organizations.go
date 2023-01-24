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
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface.
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on the groups the user has access to.
type OrganizationsClient struct {
	*clientContext
}

func (c *OrganizationsClient) Get(ctx context.Context, ref gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	project, err := c.c.GetProject(ctx, &ref.Organization)
	if err != nil {
		return nil, err
	}

	//ref.SetKey(project.Id)
	return newProject(c.clientContext, project, ref), nil
}

// List all the projects the specific user has access to.
// List returns all available projects, using multiple paginated requests if needed.

func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	apiObjs, err := c.c.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	projects := make([]gitprovider.Organization, len(apiObjs.Value))
	for i, apiObj := range apiObjs.Value {
		ref := gitprovider.OrganizationRef{
			Domain:       *apiObj.Url,
			Organization: *apiObj.Name,
		}

		teamProject := core.TeamProject{
			Id:   apiObj.Id,
			Name: apiObj.Name,
		}

		//ref.SetKey(base64.RawURLEncoding.EncodeToString(apiObj.Id))
		projects[i] = newProject(c.clientContext, &teamProject, ref)
	}
	return projects, nil

}

// Children returns the immediate child-organizations for the specific OrganizationRef o.
// The OrganizationRef may point to any existing sub-organization.
// Children returns all available organizations, using multiple paginated requests if needed.
func (c *OrganizationsClient) Children(_ context.Context, _ gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}
