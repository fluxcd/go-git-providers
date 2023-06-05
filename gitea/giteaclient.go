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
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// giteaClient is a wrapper around *gitea.Client, which implements higher-level methods,
// operating on the gitea structs.
// This interface is also fakeable, in order to unit-test the client.
type giteaClient interface {
	// Client returns the underlying *gitea.Client
	Client() *gitea.Client

	// GetOrg is a wrapper for "GET /orgs/{org}".
	// This function HTTP error wrapping, and validates the server result.
	GetOrg(ctx context.Context, orgName string) (*gitea.Organization, error)
	// ListOrgs is a wrapper for "GET /user/orgs".
	ListOrgs(ctx context.Context) ([]*gitea.Organization, error)

	// ListOrgTeamMembers is a wrapper for "GET /orgs/{org}/teams" then "GET /teams/{team}/members".
	ListOrgTeamMembers(ctx context.Context, orgName, teamName string) ([]*gitea.User, error)
	// ListOrgTeams is a wrapper for "GET /orgs/{org}/teams".
	ListOrgTeams(ctx context.Context, orgName string) ([]*gitea.Team, error)

	// GetRepo is a wrapper for "GET /repos/{owner}/{repo}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetRepo(ctx context.Context, owner, repo string) (*gitea.Repository, error)
	// ListOrgRepos is a wrapper for "GET /orgs/{org}/repos".
	ListOrgRepos(ctx context.Context, org string) ([]*gitea.Repository, error)
	// ListUserRepos is a wrapper for "GET /users/{username}/repos".
	ListUserRepos(ctx context.Context, username string) ([]*gitea.Repository, error)
	// CreateRepo is a wrapper for "POST /user/repos" (if orgName == "")
	// or "POST /orgs/{org}/repos" (if orgName != "").
	// This function handles HTTP error wrapping, and validates the server result.
	CreateRepo(ctx context.Context, orgName string, req *gitea.CreateRepoOption) (*gitea.Repository, error)
	// UpdateRepo is a wrapper for "PATCH /repos/{owner}/{repo}".
	// This function handles HTTP error wrapping, and validates the server result.
	UpdateRepo(ctx context.Context, owner, repo string, req *gitea.EditRepoOption) (*gitea.Repository, error)
	// DeleteRepo is a wrapper for "DELETE /repos/{owner}/{repo}".
	// This function handles HTTP error wrapping.
	// DANGEROUS COMMAND: In order to use this, you must set destructiveActions to true.
	DeleteRepo(ctx context.Context, owner, repo string) error

	// ListKeys is a wrapper for "GET /repos/{owner}/{repo}/keys".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListKeys(ctx context.Context, owner, repo string) ([]*gitea.DeployKey, error)
	// ListCommits is a wrapper for "GET /repos/{owner}/{repo}/git/commits".
	// This function handles pagination, HTTP error wrapping.
	ListCommits(ctx context.Context, owner, repo, branch string, perPage int, page int) ([]*gitea.Commit, error)
	// CreateCommits is a wrapper for "POST /repos/{owner}/{repo}/git/contents/{path}".
	CreateCommits(ctx context.Context, owner, repo string, path string, req *gitea.CreateFileOptions) (*gitea.FileResponse, error)
	// CreateKey is a wrapper for "POST /repos/{owner}/{repo}/keys".
	// This function handles HTTP error wrapping, and validates the server result.
	CreateKey(ctx context.Context, owner, repo string, req *gitea.DeployKey) (*gitea.DeployKey, error)
	// DeleteKey is a wrapper for "DELETE /repos/{owner}/{repo}/keys/{key_id}".
	// This function handles HTTP error wrapping.
	DeleteKey(ctx context.Context, owner, repo string, id int64) error

	// GetTeamPermissions is a wrapper for "GET /repos/{owner}/{repo}/teams/{team_slug}
	// This function handles HTTP error wrapping, and validates the server result.
	GetTeamPermissions(ctx context.Context, orgName, repo, teamName string) (*gitea.AccessMode, error)
	// ListRepoTeams is a wrapper for "GET /repos/{owner}/{repo}/teams".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListRepoTeams(ctx context.Context, orgName, repo string) ([]*gitea.Team, error)
	// AddTeam is a wrapper for "PUT /repos/{owner}/{repo}/teams/{team_slug}".
	// This function handles HTTP error wrapping.
	AddTeam(ctx context.Context, orgName, repo, teamName string, permission gitprovider.RepositoryPermission) error
	// RemoveTeam is a wrapper for "DELETE /repos/{owner}/{repo}/teams/{team_slug}".
	// This function handles HTTP error wrapping.
	RemoveTeam(ctx context.Context, orgName, repo, teamName string) error
}

type giteaClientImpl struct {
	c                  *gitea.Client
	destructiveActions bool
}

var _ giteaClient = &giteaClientImpl{}

// Client returns the underlying *gitea.Client
func (c *giteaClientImpl) Client() *gitea.Client {
	return c.c
}

// GetOrg returns a specific organization the user has access to.
func (c *giteaClientImpl) GetOrg(_ context.Context, orgName string) (*gitea.Organization, error) {
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

// ListOrgs returns all of current user's organizations.
func (c *giteaClientImpl) ListOrgs(_ context.Context) ([]*gitea.Organization, error) {
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

// ListOrgTeamMembers returns all of current team members of the given team.
func (c *giteaClientImpl) ListOrgTeamMembers(ctx context.Context, orgName, teamName string) ([]*gitea.User, error) {
	teams, err := c.ListOrgTeams(ctx, orgName)
	if err != nil {
		return nil, err
	}
	apiObjs := []*gitea.User{}
	opts := gitea.ListTeamMembersOptions{}
	for _, team := range teams {
		if team.Name == teamName {
			err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
				pageObjs, resp, listErr := c.c.ListTeamMembers(team.ID, gitea.ListTeamMembersOptions{})
				if len(pageObjs) > 0 {
					apiObjs = append(apiObjs, pageObjs...)
					return resp, listErr
				}
				return nil, nil
			})
			if err != nil {
				return nil, err
			}
			return apiObjs, nil
		}
	}

	return nil, gitprovider.ErrNotFound
}

// ListOrgTeams returns all teams of the given organization the user has access to.
func (c *giteaClientImpl) ListOrgTeams(_ context.Context, orgName string) ([]*gitea.Team, error) {
	opts := gitea.ListTeamsOptions{}
	apiObjs := []*gitea.Team{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /orgs/{org}/teams"
		pageObjs, resp, listErr := c.c.ListOrgTeams(orgName, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

// GetRepo returns the repository of the given owner by name.
func (c *giteaClientImpl) GetRepo(_ context.Context, owner, repo string) (*gitea.Repository, error) {
	apiObj, res, err := c.c.GetRepo(owner, repo)
	return validateRepositoryAPIResp(apiObj, res, err)
}

func validateRepositoryAPIResp(apiObj *gitea.Repository, res *gitea.Response, err error) (*gitea.Repository, error) {
	// If the response contained an error, return
	if err != nil {
		return nil, handleHTTPError(res, err)
	}
	// Make sure apiObj is valid
	if err := validateRepositoryAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

// ListOrgRepos returns all repositories of the given organization the user has access to.
func (c *giteaClientImpl) ListOrgRepos(_ context.Context, org string) ([]*gitea.Repository, error) {
	opts := gitea.ListOrgReposOptions{}
	apiObjs := []*gitea.Repository{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /orgs/{org}/repos
		pageObjs, resp, listErr := c.c.ListOrgRepos(org, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return validateRepositoryObjects(apiObjs)
}

func validateRepositoryObjects(apiObjs []*gitea.Repository) ([]*gitea.Repository, error) {
	for _, apiObj := range apiObjs {
		// Make sure apiObj is valid
		if err := validateRepositoryAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

// ListUserRepos returns all repositories of the given user.
func (c *giteaClientImpl) ListUserRepos(_ context.Context, username string) ([]*gitea.Repository, error) {
	opts := gitea.ListReposOptions{}
	apiObjs := []*gitea.Repository{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /users/{username}/repos
		pageObjs, resp, listErr := c.c.ListUserRepos(username, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return validateRepositoryObjects(apiObjs)
}

// CreateRepo creates a new repository for the authenticated user in the given organization.
func (c *giteaClientImpl) CreateRepo(_ context.Context, orgName string, req *gitea.CreateRepoOption) (*gitea.Repository, error) {
	if orgName != "" {
		apiObj, res, err := c.c.CreateOrgRepo(orgName, *req)
		return validateRepositoryAPIResp(apiObj, res, err)
	}
	apiObj, res, err := c.c.CreateRepo(*req)
	return validateRepositoryAPIResp(apiObj, res, err)
}

// UpdateRepo updates the given repository.
func (c *giteaClientImpl) UpdateRepo(_ context.Context, owner, repo string, req *gitea.EditRepoOption) (*gitea.Repository, error) {
	apiObj, res, err := c.c.EditRepo(owner, repo, *req)
	return validateRepositoryAPIResp(apiObj, res, err)
}

// DeleteRepo deletes the given repository.
func (c *giteaClientImpl) DeleteRepo(_ context.Context, owner, repo string) error {
	// Don't allow deleting repositories if the user didn't explicitly allow dangerous API calls.
	if !c.destructiveActions {
		return fmt.Errorf("cannot delete repository: %w", gitprovider.ErrDestructiveCallDisallowed)
	}
	resp, err := c.c.DeleteRepo(owner, repo)
	return handleHTTPError(resp, err)
}

// ListKeys returns all deploy keys of the given repository.
func (c *giteaClientImpl) ListKeys(_ context.Context, owner, repo string) ([]*gitea.DeployKey, error) {
	opts := gitea.ListDeployKeysOptions{}
	apiObjs := []*gitea.DeployKey{}

	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		// GET /repos/{owner}/{repo}/keys"
		pageObjs, resp, listErr := c.c.ListDeployKeys(owner, repo, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	for _, apiObj := range apiObjs {
		if err := validateDeployKeyAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

// ListCommits returns all commits of the given repository and branch. It uses pagination, by providing
// a page size and page number.
func (c *giteaClientImpl) ListCommits(_ context.Context, owner, repo, branch string, perPage int, page int) ([]*gitea.Commit, error) {
	opts := gitea.ListCommitOptions{
		ListOptions: gitea.ListOptions{
			PageSize: perPage,
			Page:     page,
		},
		SHA: branch,
	}
	apiObjs := []*gitea.Commit{}
	err := allPages(&opts.ListOptions, func() (*gitea.Response, error) {
		pageObjs, resp, listErr := c.c.ListRepoCommits(owner, repo, opts)
		if len(pageObjs) > 0 {
			apiObjs = append(apiObjs, pageObjs...)
			return resp, listErr
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	return apiObjs, nil
}

// CreateCommits creates a new commit for the given repository.
func (c *giteaClientImpl) CreateCommits(_ context.Context, owner, repo string, path string, req *gitea.CreateFileOptions) (*gitea.FileResponse, error) {
	apiObj, res, err := c.c.CreateFile(owner, repo, path, *req)
	if err != nil {
		return nil, handleHTTPError(res, err)
	}
	return apiObj, nil
}

// CreateKey creates a new deploy key for the given repository.
func (c *giteaClientImpl) CreateKey(_ context.Context, owner, repo string, req *gitea.DeployKey) (*gitea.DeployKey, error) {
	opts := gitea.CreateKeyOption{Title: req.Title, Key: req.Key, ReadOnly: req.ReadOnly}
	apiObj, resp, err := c.c.CreateDeployKey(owner, repo, opts)
	if err != nil {
		return nil, handleHTTPError(resp, err)
	}
	if err := validateDeployKeyAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

// DeleteKey deletes the given deploy key from the given repository.
func (c *giteaClientImpl) DeleteKey(_ context.Context, owner, repo string, id int64) error {
	res, err := c.c.DeleteDeployKey(owner, repo, id)
	return handleHTTPError(res, err)
}

// GetTeamPermissions returns the permissions of the given team on the given repository.
func (c *giteaClientImpl) GetTeamPermissions(_ context.Context, orgName, repo, teamName string) (*gitea.AccessMode, error) {
	apiObj, resp, err := c.c.CheckRepoTeam(orgName, repo, teamName)
	if err != nil {
		return nil, handleHTTPError(resp, err)
	}
	if apiObj == nil {
		return nil, fmt.Errorf("team %s not found in repository %s/%s", teamName, orgName, repo)
	}

	return &apiObj.Permission, nil
}

// ListRepoTeams returns all teams of the given repository.
func (c *giteaClientImpl) ListRepoTeams(ctx context.Context, orgName, repo string) ([]*gitea.Team, error) {
	teamObjs, resp, err := c.c.GetRepoTeams(orgName, repo)
	if err != nil {
		return nil, handleHTTPError(resp, err)
	}
	return teamObjs, nil
}

// AddTeam adds the given team to the given repository.
// We don't support setting permissions for Gitea, so we ignore the permission parameter.
// see https://github.com/go-gitea/gitea/issues/14717
func (c *giteaClientImpl) AddTeam(_ context.Context, orgName, repo, teamName string, permission gitprovider.RepositoryPermission) error {
	res, err := c.c.AddRepoTeam(orgName, repo, teamName)
	return handleHTTPError(res, err)
}

// RemoveTeam removes the given team from the given repository.
func (c *giteaClientImpl) RemoveTeam(_ context.Context, orgName, repo, teamName string) error {
	res, err := c.c.RemoveRepoTeam(orgName, repo, teamName)
	return handleHTTPError(res, err)
}
