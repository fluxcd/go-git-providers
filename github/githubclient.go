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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/google/go-github/v71/github"
)

// githubClientImpl is a wrapper around *github.Client, which implements higher-level methods,
// operating on the go-github structs. Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
// This interface is also fakeable, in order to unit-test the client.
type githubClient interface {
	// Client returns the underlying *github.Client
	Client() *github.Client

	// GetOrg is a wrapper for "GET /orgs/{org}".
	// This function HTTP error wrapping, and validates the server result.
	GetOrg(ctx context.Context, orgName string) (*github.Organization, error)
	// ListOrgs is a wrapper for "GET /user/orgs".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListOrgs(ctx context.Context) ([]*github.Organization, error)

	// ListOrgTeamMembers is a wrapper for "GET /orgs/{org}/teams/{team_slug}/members".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListOrgTeamMembers(ctx context.Context, orgName, teamName string) ([]*github.User, error)
	// ListOrgTeams is a wrapper for "GET /orgs/{org}/teams".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListOrgTeams(ctx context.Context, orgName string) ([]*github.Team, error)

	// GetRepo is a wrapper for "GET /repos/{owner}/{repo}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetRepo(ctx context.Context, owner, repo string) (*github.Repository, error)
	// ListOrgRepos is a wrapper for "GET /orgs/{org}/repos".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListOrgRepos(ctx context.Context, org string) ([]*github.Repository, error)
	// ListUserRepos is a wrapper for "GET /users/{username}/repos".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListUserRepos(ctx context.Context, username string) ([]*github.Repository, error)
	// CreateRepo is a wrapper for "POST /user/repos" (if orgName == "")
	// or "POST /orgs/{org}/repos" (if orgName != "").
	// This function handles HTTP error wrapping, and validates the server result.
	CreateRepo(ctx context.Context, orgName string, req *github.Repository) (*github.Repository, error)
	// UpdateRepo is a wrapper for "PATCH /repos/{owner}/{repo}".
	// This function handles HTTP error wrapping, and validates the server result.
	UpdateRepo(ctx context.Context, owner, repo string, req *github.Repository) (*github.Repository, error)
	// DeleteRepo is a wrapper for "DELETE /repos/{owner}/{repo}".
	// This function handles HTTP error wrapping.
	// DANGEROUS COMMAND: In order to use this, you must set destructiveActions to true.
	DeleteRepo(ctx context.Context, owner, repo string) error

	// GetUser is a wrapper for "GET /user"
	GetUser(ctx context.Context) (*github.User, error)

	// ListKeys is a wrapper for "GET /repos/{owner}/{repo}/keys".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListKeys(ctx context.Context, owner, repo string) ([]*github.Key, error)
	// ListCommitsPage is a wrapper for "GET /repos/{owner}/{repo}/commits".
	// This function handles pagination, HTTP error wrapping.
	ListCommitsPage(ctx context.Context, owner, repo, branch string, perPage int, page int) ([]*github.Commit, error)
	// CreateKey is a wrapper for "POST /repos/{owner}/{repo}/keys".
	// This function handles HTTP error wrapping, and validates the server result.
	CreateKey(ctx context.Context, owner, repo string, req *github.Key) (*github.Key, error)
	// DeleteKey is a wrapper for "DELETE /repos/{owner}/{repo}/keys/{key_id}".
	// This function handles HTTP error wrapping.
	DeleteKey(ctx context.Context, owner, repo string, id int64) error

	// GetTeamPermissions is a wrapper for "GET /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetTeamPermissions(ctx context.Context, orgName, repo, teamName string) (map[string]bool, error)
	// ListRepoTeams is a wrapper for "GET /repos/{owner}/{repo}/teams".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListRepoTeams(ctx context.Context, orgName, repo string) ([]*github.Team, error)
	// AddTeam is a wrapper for "PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}".
	// This function handles HTTP error wrapping.
	AddTeam(ctx context.Context, orgName, repo, teamName string, permission gitprovider.RepositoryPermission) error
	// RemoveTeam is a wrapper for "DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}".
	// This function handles HTTP error wrapping.
	RemoveTeam(ctx context.Context, orgName, repo, teamName string) error
}

// githubClientImpl is a wrapper around *github.Client, which implements higher-level methods,
// operating on the go-github structs. See the githubClient interface for method documentation.
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
type githubClientImpl struct {
	c                  *github.Client
	destructiveActions bool
}

// githubClientImpl implements githubClient.
var _ githubClient = &githubClientImpl{}

func (c *githubClientImpl) Client() *github.Client {
	return c.c
}

func (c *githubClientImpl) GetOrg(ctx context.Context, orgName string) (*github.Organization, error) {
	// GET /orgs/{org}
	apiObj, _, err := c.c.Organizations.Get(ctx, orgName)
	if err != nil {
		return nil, handleHTTPError(err)
	}
	// Validate the API object
	if err := validateOrganizationAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *githubClientImpl) ListOrgs(ctx context.Context) ([]*github.Organization, error) {
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

	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateOrganizationAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *githubClientImpl) ListOrgTeamMembers(ctx context.Context, orgName, teamName string) ([]*github.User, error) {
	apiObjs := []*github.User{}
	opts := &github.TeamListTeamMembersOptions{}
	err := allPages(&opts.ListOptions, func() (*github.Response, error) {
		// GET /orgs/{org}/teams/{team_slug}/members
		pageObjs, resp, listErr := c.c.Teams.ListTeamMembersBySlug(ctx, orgName, teamName, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}

	// Make sure the Login field is set.
	for _, apiObj := range apiObjs {
		if apiObj.Login == nil {
			return nil, fmt.Errorf("didn't expect login to be nil for user: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
		}
	}

	return apiObjs, nil
}

func (c *githubClientImpl) ListOrgTeams(ctx context.Context, orgName string) ([]*github.Team, error) {
	// List all teams, using pagination. This does not contain information about the members
	apiObjs := []*github.Team{}
	opts := &github.ListOptions{}
	err := allPages(opts, func() (*github.Response, error) {
		// GET /orgs/{org}/teams
		pageObjs, resp, listErr := c.c.Teams.ListTeams(ctx, orgName, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}

	// Make sure the Slug field is set.
	for _, apiObj := range apiObjs {
		if apiObj.Slug == nil {
			return nil, fmt.Errorf("didn't expect slug to be nil for team: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
		}
	}
	return apiObjs, nil
}

func (c *githubClientImpl) GetRepo(ctx context.Context, owner, repo string) (*github.Repository, error) {
	// GET /repos/{owner}/{repo}
	apiObj, _, err := c.c.Repositories.Get(ctx, owner, repo)
	return validateRepositoryAPIResp(apiObj, err)
}

func validateRepositoryAPIResp(apiObj *github.Repository, err error) (*github.Repository, error) {
	// If the response contained an error, return
	if err != nil {
		return nil, handleHTTPError(err)
	}
	// Make sure apiObj is valid
	if err := validateRepositoryAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *githubClientImpl) ListOrgRepos(ctx context.Context, org string) ([]*github.Repository, error) {
	var apiObjs []*github.Repository
	opts := &github.RepositoryListByOrgOptions{}
	err := allPages(&opts.ListOptions, func() (*github.Response, error) {
		// GET /orgs/{org}/repos
		pageObjs, resp, listErr := c.c.Repositories.ListByOrg(ctx, org, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}
	return validateRepositoryObjects(apiObjs)
}

func validateRepositoryObjects(apiObjs []*github.Repository) ([]*github.Repository, error) {
	for _, apiObj := range apiObjs {
		// Make sure apiObj is valid
		if err := validateRepositoryAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *githubClientImpl) ListUserRepos(ctx context.Context, username string) ([]*github.Repository, error) {
	var apiObjs []*github.Repository
	opts := &github.RepositoryListOptions{}
	err := allPages(&opts.ListOptions, func() (*github.Response, error) {
		// GET /users/{username}/repos
		pageObjs, resp, listErr := c.c.Repositories.List(ctx, username, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}
	return validateRepositoryObjects(apiObjs)
}

func (c *githubClientImpl) CreateRepo(ctx context.Context, orgName string, req *github.Repository) (*github.Repository, error) {
	// POST /user/repos (if orgName == "")
	// POST /orgs/{org}/repos (if orgName != "")
	setPrivate := true
	if *req.Visibility == "private" {
		req.Private = &setPrivate
	}
	apiObj, _, err := c.c.Repositories.Create(ctx, orgName, req)
	return validateRepositoryAPIResp(apiObj, err)
}

func (c *githubClientImpl) UpdateRepo(ctx context.Context, owner, repo string, req *github.Repository) (*github.Repository, error) {
	// PATCH /repos/{owner}/{repo}
	apiObj, _, err := c.c.Repositories.Edit(ctx, owner, repo, req)
	return validateRepositoryAPIResp(apiObj, err)
}

func (c *githubClientImpl) DeleteRepo(ctx context.Context, owner, repo string) error {
	// Don't allow deleting repositories if the user didn't explicitly allow dangerous API calls.
	if !c.destructiveActions {
		return fmt.Errorf("cannot delete repository: %w", gitprovider.ErrDestructiveCallDisallowed)
	}
	// DELETE /repos/{owner}/{repo}
	_, err := c.c.Repositories.Delete(ctx, owner, repo)
	return handleHTTPError(err)
}

func (c *githubClientImpl) ListKeys(ctx context.Context, owner, repo string) ([]*github.Key, error) {
	apiObjs := []*github.Key{}
	opts := &github.ListOptions{}
	err := allPages(opts, func() (*github.Response, error) {
		// GET /repos/{owner}/{repo}/keys
		pageObjs, resp, listErr := c.c.Repositories.ListKeys(ctx, owner, repo, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
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

func (c *githubClientImpl) GetUser(ctx context.Context) (*github.User, error) {
	// GET /user
	user, _, err := c.c.Users.Get(ctx, "")
	return user, err
}

func (c *githubClientImpl) ListCommitsPage(ctx context.Context, owner, repo, branch string, perPage int, page int) ([]*github.Commit, error) {
	apiObjs := make([]*github.Commit, 0)
	lcOpts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
		SHA: branch,
	}

	// GET /repos/{owner}/{repo}/commits
	pageObjs, _, listErr := c.c.Repositories.ListCommits(ctx, owner, repo, lcOpts)
	for _, c := range pageObjs {
		apiObjs = append(apiObjs, &github.Commit{
			SHA: c.SHA,
			Tree: &github.Tree{
				SHA: c.Commit.Tree.SHA,
			},
			Author:  c.Commit.Author,
			Message: c.Commit.Message,
			URL:     c.HTMLURL,
		})
	}

	if listErr != nil {
		return nil, listErr
	}
	return apiObjs, nil
}

func (c *githubClientImpl) CreateKey(ctx context.Context, owner, repo string, req *github.Key) (*github.Key, error) {
	// POST /repos/{owner}/{repo}/keys
	apiObj, _, err := c.c.Repositories.CreateKey(ctx, owner, repo, req)
	if err != nil {
		return nil, handleHTTPError(err)
	}
	if err := validateDeployKeyAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *githubClientImpl) DeleteKey(ctx context.Context, owner, repo string, id int64) error {
	// DELETE /repos/{owner}/{repo}/keys/{key_id}
	_, err := c.c.Repositories.DeleteKey(ctx, owner, repo, id)
	return handleHTTPError(err)
}

func (c *githubClientImpl) GetTeamPermissions(ctx context.Context, orgName, repo, teamName string) (map[string]bool, error) {
	// GET /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	apiObj, _, err := c.c.Teams.IsTeamRepoBySlug(ctx, orgName, teamName, orgName, repo)
	if err != nil {
		return nil, handleHTTPError(err)
	}

	// Make sure permissions isn't nil
	if apiObj.Permissions == nil {
		return nil, fmt.Errorf("didn't expect permissions to be nil for team: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
	}
	return apiObj.Permissions, nil
}

func (c *githubClientImpl) ListRepoTeams(ctx context.Context, orgName, repo string) ([]*github.Team, error) {
	apiObjs := []*github.Team{}
	opts := &github.ListOptions{}
	err := allPages(opts, func() (*github.Response, error) {
		// GET /repos/{owner}/{repo}/teams
		pageObjs, resp, listErr := c.c.Repositories.ListTeams(ctx, orgName, repo, opts)
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}

	// Make sure the Slug field isn't nil
	for _, apiObj := range apiObjs {
		if apiObj.Slug == nil {
			return nil, fmt.Errorf("didn't expect slug to be nil for team: %+v: %w", apiObj, gitprovider.ErrInvalidServerData)
		}
	}
	return apiObjs, nil
}

func (c *githubClientImpl) AddTeam(ctx context.Context, orgName, repo, teamName string, permission gitprovider.RepositoryPermission) error {
	// PUT /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	_, err := c.c.Teams.AddTeamRepoBySlug(ctx, orgName, teamName, orgName, repo, &github.TeamAddTeamRepoOptions{
		Permission: string(permission),
	})
	return handleHTTPError(err)
}

func (c *githubClientImpl) RemoveTeam(ctx context.Context, orgName, repo, teamName string) error {
	// DELETE /orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
	_, err := c.c.Teams.RemoveTeamRepoBySlug(ctx, orgName, teamName, orgName, repo)
	return handleHTTPError(err)
}
