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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v6/core"
)

// OrganizationsClient implements the gitprovider.OrganizationsClient interface.
var _ gitprovider.OrganizationsClient = &OrganizationsClient{}

// OrganizationsClient operates on the projects a user has access to.
type OrganizationsClient struct {
	*clientContext
}

// Get returns the specific project the user has access to.
func (c *OrganizationsClient) Get(ctx context.Context, ref gitprovider.OrganizationRef) (gitprovider.Organization, error) {
	args := core.GetProjectArgs{ProjectId: &ref.Organization}
	// https://pkg.go.dev/github.com/microsoft/azure-devops-go-api/azuredevops/v6@v6.0.1/core#ClientImpl.GetProject
	project, err := c.c.GetProject(ctx, args)
	if err != nil {
		return nil, err
	}

	ref.SetKey(project.Id.String())
	return newProject(c.clientContext, *project, ref), nil
}

// List returns all available projects, using multiple requests if needed.
func (c *OrganizationsClient) List(ctx context.Context) ([]gitprovider.Organization, error) {
	args := core.GetProjectsArgs{}
	projects := make([]gitprovider.Organization, 0)

	return c.list(ctx, args, projects)
}

func (c *OrganizationsClient) list(ctx context.Context, args core.GetProjectsArgs, projects []gitprovider.Organization) ([]gitprovider.Organization, error) {
	// https://pkg.go.dev/github.com/microsoft/azure-devops-go-api/azuredevops/v6@v6.0.1/core#ClientImpl.GetProjects
	apiObjs, err := c.c.GetProjects(ctx, args)
	if err != nil {
		return nil, err
	}

	for _, apiObj := range apiObjs.Value {
		ref := gitprovider.OrganizationRef{
			Domain:       *apiObj.Url,
			Organization: *apiObj.Name,
		}

		teamProject := core.TeamProject{
			Id:   apiObj.Id,
			Name: apiObj.Name,
		}

		ref.SetKey(apiObj.Id.String())
		projects = append(projects, newProject(c.clientContext, teamProject, ref))
	}

	if apiObjs.ContinuationToken != "" {
		args := core.GetProjectsArgs{
			ContinuationToken: &apiObjs.ContinuationToken,
		}
		return c.list(ctx, args, projects)
	}

	return projects, nil
}

// Children is not supported by AzureDevops.
func (c *OrganizationsClient) Children(_ context.Context, _ gitprovider.OrganizationRef) ([]gitprovider.Organization, error) {
	return nil, gitprovider.ErrNoProviderSupport
}
