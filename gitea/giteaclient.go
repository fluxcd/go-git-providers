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

package gitea

import (
	"context"
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

// giteaClientImpl is a wrapper around *gitea.Client, which implements higher-level methods,
// operating on the gitea structs. TODO: Verify pagination is implemented for all List* methods,
// all returnedobjects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
// This interface is also fakeable, in order to unit-test the client.
type giteaClient interface {
	// Client returns the underlying *gitea.Client
	Client() *gitea.Client

	GetOrg(ctx context.Context, orgName string) (*gitea.Organization, error)

	ListOrgs(ctx context.Context) ([]*gitea.Organization, error)

	ListOrgTeamMembers(ctx context.Context, orgName, teamName string) ([]*gitea.User, error)

	ListOrgTeams(ctx context.Context, orgName string) ([]*gitea.Team, error)

	GetRepo(ctx context.Context, owner, repo string) (*gitea.Repository, error)

	ListOrgRepos(ctx context.Context, org string) ([]*gitea.Repository, error)

	ListUserRepos(ctx context.Context, username string) ([]*gitea.Repository, error)

	CreateRepo(ctx context.Context, orgName string, req *gitea.CreateRepoOption) (*gitea.Repository, error)

	UpdateRepo(ctx context.Context, owner, repo string, req *gitea.EditRepoOption) (*gitea.Repository, error)
	// DANGEROUS COMMAND: In order to use this, you must set destructiveActions to true.
	DeleteRepo(ctx context.Context, owner, repo string) error

	ListKeys(ctx context.Context, owner, repo string) ([]*gitea.DeployKey, error)

	ListCommitsPage(ctx context.Context, owner, repo, branch string, perPage int, page int) ([]*gitea.Commit, error)

	CreateKey(ctx context.Context, owner, repo string, req *gitea.DeployKey) (*gitea.DeployKey, error)

	DeleteKey(ctx context.Context, owner, repo string, id int64) error

	GetTeamPermissions(ctx context.Context, orgName, repo, teamName string) (*gitea.AccessMode, error)

	ListRepoTeams(ctx context.Context, orgName, repo string) ([]*gitea.Team, error)

	AddTeam(ctx context.Context, orgName, repo, teamName string, permission gitprovider.RepositoryPermission) error

	RemoveTeam(ctx context.Context, orgName, repo, teamName string) error
}

type giteaClientImpl struct {
	c                  *gitea.Client
	destructiveActions bool
}

var _ giteaClient = &giteaClientImpl{}

func (c *giteaClientImpl) Client() *gitea.Client {
	return c.c
}

func (c *giteaClientImpl) GetOrg(ctx context.Context, orgName string) (*gitea.Organization, error) {
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

func (c *giteaClientImpl) ListOrgs(ctx context.Context) ([]*gitea.Organization, error) {
	opts := gitea.ListOrgsOptions{}
	apiObjs, _, err := c.c.ListMyOrgs(opts)

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

func (c *giteaClientImpl) ListOrgTeamMembers(ctx context.Context, orgName, teamName string) ([]*gitea.User, error) {
	orgTeamOpts := gitea.ListTeamsOptions{}
	teams, _, err := c.c.ListOrgTeams(orgName, orgTeamOpts)
	if err != nil {
		return nil, err
	}
	apiObjs := []*gitea.User{}
	for _, team := range teams {
		users, _, err := c.c.ListTeamMembers(team.ID, gitea.ListTeamMembersOptions{})
		if err != nil {
			continue
		}
		apiObjs = append(apiObjs, users...)
	}

	return apiObjs, nil
}

func (c *giteaClientImpl) ListOrgTeams(ctx context.Context, orgName string) ([]*gitea.Team, error) {
	orgTeamOpts := gitea.ListTeamsOptions{}
	apiObjs, _, err := c.c.ListOrgTeams(orgName, orgTeamOpts)
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

func (c *giteaClientImpl) GetRepo(ctx context.Context, owner, repo string) (*gitea.Repository, error) {
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

func (c *giteaClientImpl) ListOrgRepos(ctx context.Context, org string) ([]*gitea.Repository, error) {
	opts := gitea.ListOrgReposOptions{}
	apiObjs, _, err := c.c.ListOrgRepos(org, opts)
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

func (c *giteaClientImpl) ListUserRepos(ctx context.Context, username string) ([]*gitea.Repository, error) {
	opts := gitea.ListReposOptions{}
	apiObjs, _, err := c.c.ListUserRepos(username, opts)
	if err != nil {
		return nil, err
	}
	return validateRepositoryObjects(apiObjs)
}

func (c *giteaClientImpl) CreateRepo(ctx context.Context, orgName string, req *gitea.CreateRepoOption) (*gitea.Repository, error) {
	if orgName != "" {
		apiObj, res, err := c.c.CreateOrgRepo(orgName, *req)
		return validateRepositoryAPIResp(apiObj, res, err)
	}
	apiObj, res, err := c.c.CreateRepo(*req)
	return validateRepositoryAPIResp(apiObj, res, err)
}

func (c *giteaClientImpl) UpdateRepo(ctx context.Context, owner, repo string, req *gitea.EditRepoOption) (*gitea.Repository, error) {
	apiObj, res, err := c.c.EditRepo(owner, repo, *req)
	return validateRepositoryAPIResp(apiObj, res, err)
}

func (c *giteaClientImpl) DeleteRepo(ctx context.Context, owner, repo string) error {
	// Don't allow deleting repositories if the user didn't explicitly allow dangerous API calls.
	if !c.destructiveActions {
		return fmt.Errorf("cannot delete repository: %w", gitprovider.ErrDestructiveCallDisallowed)
	}
	res, err := c.c.DeleteRepo(owner, repo)
	return handleHTTPError(res, err)
}

func (c *giteaClientImpl) ListKeys(ctx context.Context, owner, repo string) ([]*gitea.DeployKey, error) {
	opts := gitea.ListDeployKeysOptions{}
	apiObjs, _, err := c.c.ListDeployKeys(owner, repo, opts)
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

func (c *giteaClientImpl) ListCommitsPage(ctx context.Context, owner, repo, branch string, perPage int, page int) ([]*gitea.Commit, error) {
	opts := gitea.ListCommitOptions{
		ListOptions: gitea.ListOptions{
			PageSize: perPage,
			Page:     page,
		},
		SHA: branch,
	}
	apiObjs, _, listErr := c.c.ListRepoCommits(owner, repo, opts)

	if listErr != nil {
		return nil, listErr
	}
	return apiObjs, nil
}

func (c *giteaClientImpl) CreateKey(ctx context.Context, owner, repo string, req *gitea.DeployKey) (*gitea.DeployKey, error) {
	opts := gitea.CreateKeyOption{Title: req.Title, Key: req.Key, ReadOnly: req.ReadOnly}
	apiObj, res, err := c.c.CreateDeployKey(owner, repo, opts)
	if err != nil {
		return nil, handleHTTPError(res, err)
	}
	if err := validateDeployKeyAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *giteaClientImpl) DeleteKey(ctx context.Context, owner, repo string, id int64) error {
	res, err := c.c.DeleteDeployKey(owner, repo, id)
	return handleHTTPError(res, err)
}

func (c *giteaClientImpl) GetTeamPermissions(ctx context.Context, orgName, repo, teamName string) (*gitea.AccessMode, error) {
	apiObj, res, err := c.c.CheckRepoTeam(orgName, repo, teamName)
	if err != nil {
		return nil, handleHTTPError(res, err)
	}

	return &apiObj.Permission, nil
}

func (c *giteaClientImpl) ListRepoTeams(ctx context.Context, orgName, repo string) ([]*gitea.Team, error) {
	opts := gitea.ListTeamsOptions{}
	teamObjs, _, err := c.c.ListOrgTeams(orgName, opts)
	if err != nil {
		return nil, err
	}
	apiObjs := []*gitea.Team{}
	for _, teamObj := range teamObjs {
		teamRepo, _, checkErr := c.c.CheckRepoTeam(orgName, repo, teamObj.Name)
		if checkErr != nil {
			continue
		}
		if teamRepo != nil {
			apiObjs = append(apiObjs, teamRepo)
		}
	}
	return apiObjs, nil
}

func (c *giteaClientImpl) AddTeam(ctx context.Context, orgName, repo, teamName string, permission gitprovider.RepositoryPermission) error {
	res, err := c.c.AddRepoTeam(orgName, repo, teamName)
	return handleHTTPError(res, err)
}

func (c *giteaClientImpl) RemoveTeam(ctx context.Context, orgName, repo, teamName string) error {
	res, err := c.c.RemoveRepoTeam(orgName, repo, teamName)
	return handleHTTPError(res, err)
}
