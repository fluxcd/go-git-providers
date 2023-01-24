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
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
)

type azureDevopsClient interface {
	// Client core.Client provides access to the main core Azure Devops APIS
	Client() core.Client
	//  git.Client provides access to the git repository resource for Azure Devops APIS
	gitClient() git.Client
	// ListProjects get list of Azure Devops projects in an organization
	// This is a wrapper for  "Get projects/"
	ListProjects(ctx context.Context) (*core.GetProjectsResponseValue, error)
	// GetProject retrieves a selected project
	// This is a wrapper for "GET projects/{projectId} "
	GetProject(ctx context.Context, projectName *string) (*core.TeamProject, error)
	// GetRepo is a wrapper for "GET /{project}/_apis/git/repositories/{repositoryId}".

	// This function handles HTTP error wrapping, and validates the server result.
	GetRepo(ctx context.Context, project, repo string) (git.GitRepository, error)
	ListRepos(ctx context.Context, org string) ([]*git.GitRepository, error)
}

// azureDevopsClientImpl is a wrapper around *azureDevops.Client, which implements higher-level methods,
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
type azureDevopsClientImpl struct {
	c                  core.Client
	g                  git.Client
	destructiveActions bool
}

func (c *azureDevopsClientImpl) ListRepos(ctx context.Context, org string) ([]*git.GitRepository, error) {
	//TODO implement me
	panic("implement me")
}

func (c *azureDevopsClientImpl) GetRepo(ctx context.Context, project, repo string) (git.GitRepository, error) {
	opts := git.GetRepositoryArgs{RepositoryId: &repo, Project: &project}
	apiObj, err := c.g.GetRepository(ctx, opts)
	return *apiObj, err
}

func (c *azureDevopsClientImpl) gitClient() git.Client {
	return c.g
}

var _ azureDevopsClient = &azureDevopsClientImpl{}

func (c *azureDevopsClientImpl) Client() core.Client {
	return c.c
}

func (c *azureDevopsClientImpl) ListProjects(ctx context.Context) (*core.GetProjectsResponseValue, error) {
	opts := core.GetProjectsArgs{}

	apiObj, err := c.c.GetProjects(ctx, opts)
	if err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *azureDevopsClientImpl) GetProject(ctx context.Context, projectName *string) (*core.TeamProject, error) {
	opts := core.GetProjectArgs{ProjectId: projectName}
	apiObj, err := c.c.GetProject(ctx, opts)
	return apiObj, err
}

func (c *azureDevopsClientImpl) ListPullRequests(ctx context.Context, repositoryId *string) ([]git.GitPullRequest, error) {
	apiObj, err := c.g.GetPullRequests(ctx, git.GetPullRequestsArgs{RepositoryId: repositoryId})
	if err != nil {
		return nil, err
	}
	return *apiObj, nil
}
