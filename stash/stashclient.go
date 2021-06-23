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

package stash

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
)

// stashClientImpl is a wrapper around httpclient.ReqResp, which implements rest API access,
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
// This interface is also fakeable, in order to unit-test the client.
type stashClient interface {
	// Client returns the underlying httpclient.ReqResp
	Client() httpclient.ReqResp

	// GetUser returns the user details for a given user name.
	GetUser(ctx context.Context, user string) (*User, error)

	// Group methods

	// GetGroup is a wrapper for "GET /rest/api/1.0/admin/groups?filter={group}".
	// This function HTTP error wrapping, and validates the server result.
	GetGroup(ctx context.Context, groupID interface{}) (*Group, error)

	// ListGroups is a wrapper for "GET /rest/api/1.0/admin/groups".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListGroups(ctx context.Context) ([]*Group, error)

	// ListGroupMembers is a wrapper for "GET /rest/api/1.0/admin/groups/more-members?context={group}".
	// It retruns the users who are members of a group/project
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListGroupMembers(ctx context.Context, groupID interface{}) ([]*User, error)

	// GetGroupMembers is a wrapper for "GET /rest/api/1.0/admin/groups/more-members?context={group}&filter={user}".
	// It returns the user if a member of a group/project or nil if not
	GetGroupMember(ctx context.Context, groupID interface{}, userID interface{}) (*GroupMembers, error)

	// Project methods

	// GetProject is a wrapper for "GET /rest/api/1.0/projects?filter={project}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetProject(ctx context.Context, projectName string) (*Project, error)

	// ListProjects is a wrapper for "GET /rest/api/1.0/projects".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListProjects(ctx context.Context) ([]*Project, error)

	// ListProjectGroups is a wrapper for "GET /rest/api/1.0/projects/permissions/groups".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListProjectGroups(ctx context.Context, projectName string) ([]*ProjectGroupPermission, error)

	// Repository methods

	// GetRepositoryTeamPermissions returns the permissions a team has on a repository
	// returns empty string if no permissions
	GetRepositoryTeamPermissions(ctx context.Context, orgName, repo, teamName string) (string, error)

	// ListRepositoryTeamPermissions returns the permissions each team has on a repository
	// returns empty list if no permissions
	ListRepositoryTeamPermissions(ctx context.Context, orgName, repo string) ([]*RepositoryGroupPermission, error)

	// CreateRepositoryTeamPermissions adds a group permission to a repo
	CreateRepositoryTeamPermissions(ctx context.Context, orgName, repo, teamName, permission string) error

	// GetRepository is a wrapper for "GET /rest/api/1.0/projects/{project|user}/repos/{repoID}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetRepository(ctx context.Context, owner, name string) (*Repository, error)

	// ListRepositories is a wrapper for "GET /rest/api/1.0/projects/{project|user}/repos".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListRepositories(ctx context.Context, owner string) ([]*Repository, error)

	// CreateRepository is a wrapper for "POST /rest/api/1.0/projects/{project|user}/repos/{repoID}"
	// This function handles HTTP error wrapping, and validates the server result.
	CreateRepository(ctx context.Context, owner string, req *Repository) (*Repository, error)

	// UpdateRepository is a wrapper for "PUT /rest/api/1.0/projects/{project|user}/repos/{repoID}".
	// This function handles HTTP error wrapping, and validates the server result.
	UpdateRepository(ctx context.Context, owner string, req *Repository) (*Repository, error)

	// DeleteProject is a wrapper for "DELETE /rest/api/1.0/projects/{project|user}/repos/{repoID}".
	// This function handles HTTP error wrapping.
	// DANGEROUS COMMAND: In order to use this, you must set destructiveActions to true.
	DeleteRepository(ctx context.Context, owner, name string) error

	// Deploy key methods

	// ListKeys is a wrapper for "GET /rest/keys/1.0/projects/{project|user}/repos/{repoID}/ssh".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListKeys(ctx context.Context, owner, repoName string) ([]*DeployKey, error)

	// CreateProjectKey is a wrapper for "POST /rest/keys/1.0/projects/{project|user}/repos/{repoID}/ssh/{key_id}".
	// This function handles HTTP error wrapping, and validates the server result.
	CreateKey(ctx context.Context, req *DeployKey) (*DeployKey, error)

	// DeleteKey is a wrapper for "DELETE /rest/keys/1.0/projects/{project|user}/repos/{repoID}/ssh/{key_id}".
	// This function handles HTTP error wrapping.
	DeleteKey(ctx context.Context, owner, repoName string, keyID int) error

	// UpdateKey is a wrapper for "PUT /rest/keys/1.0/projects/{project|user}/repos/{repoID}/ssh/{key_id}/permission/{permission}".
	// This function handles HTTP error wrapping.
	UpdateKey(ctx context.Context, owner, repoName string, keyID int, permission string) (*DeployKey, error)

	// Pull Request methods

	// ListPullRequests is a wrapper for "GET /rest/api/1.0/projects/{project|user}/repos/{repoID}/pull-requests".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListPullRequests(ctx context.Context, owner, repoName string) ([]*PullRequest, error)

	// GetPullRequest is a wrapper for "GET /rest/api/1.0/projects/{project|user}/repos/{repoID}/pull-requests".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	GetPullRequest(ctx context.Context, owner, repoName string, prID int) (*PullRequest, error)

	// CreatePullRequest is a wrapper for "POST /rest/api/1.0/projects/{project|user}/repos/{repoID}/pull-requests".
	// This function handles HTTP error wrapping, and validates the server result.
	CreatePullRequest(ctx context.Context, owner, repoName string, pr *PullRequestCreation) (*PullRequest, error)

	// DeletePullRequest is a wrapper for "DELETE /rest/api/1.0/projects/{project|user}/repos/{repoID}/pull-requests".
	// This function handles HTTP error wrapping.
	DeletePullRequest(ctx context.Context, owner, repoName string, prID int) error

	// UpdatePullRequest is a wrapper for "PUT /rest/api/1.0/projects/{project|user}/repos/{repoID}/pull-requests".
	// This function handles HTTP error wrapping.
	UpdatePullRequest(ctx context.Context, owner, repoName string, pr *PullRequest) (*PullRequest, error)

	// Commits

	// ListCommitsPage is a wrapper for "GET /projects/{project}/repos/{repo}/commits".
	// This function handles pagination, HTTP error wrapping.
	ListCommitsPage(ctx context.Context, owner, repoName, branch string, perPage int, page int) ([]*Commit, error)

	// GetCommit gets a commit
	GetCommit(ctx context.Context, owner, repoName, commitID string) (*Commit, error)

	// CreateBranch creates a branch
	CreateBranch(ctx context.Context, owner, repoName, branch string) error

	// getOwnerID retruns the project slug. If the project name is not found it returns the name with tilde prefix to be used as a user name.
	getOwnerID(ctx context.Context, projectName string) string

	//getLogger gets the logger
	getLogger() logr.Logger
}

// stashClientImpl is a wrapper around httpclient.ReqResp and Client
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
type stashClientImpl struct {
	c                  httpclient.ReqResp
	destructiveActions bool
	log                logr.Logger
}

// stashClientImpl implements stashClient.
var _ stashClient = &stashClientImpl{}

func (c *stashClientImpl) Client() httpclient.ReqResp {
	return c.c
}

func (c *stashClientImpl) getLogger() logr.Logger {
	return c.log
}

func (c *stashClientImpl) getOwnerID(ctx context.Context, projectName string) string {
	if len(projectName) > 0 && projectName[0] == '~' {
		return projectName
	}
	project, err := c.GetProject(ctx, projectName)
	if err != nil {
		return addTilde(projectName)
	}
	return project.Key
}

func (c *stashClientImpl) GetUser(ctx context.Context, userName string) (*User, error) {
	users := NewStashUsers(c)
	user, err := users.Get(ctx, userName)
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	if err := validateUserAPI(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (c *stashClientImpl) GetGroup(ctx context.Context, groupID interface{}) (*Group, error) {
	groups := NewStashGroups(c)
	group, err := groups.Get(ctx, groupID.(string))
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	if err := validateGroupAPI(group); err != nil {
		return nil, err
	}

	return group, nil
}

func (c *stashClientImpl) ListGroups(ctx context.Context) ([]*Group, error) {
	groups := NewStashGroups(c)
	apiObjs := []*Group{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /groups
		paging, listErr := groups.List(ctx, opts)
		apiObjs = append(apiObjs, groups.getGroups()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateGroupAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) GetGroupMember(ctx context.Context, groupID interface{}, userID interface{}) (*GroupMembers, error) {
	return nil, gitprovider.ErrNoProviderSupport
}

func (c *stashClientImpl) ListGroupMembers(ctx context.Context, groupID interface{}) ([]*User, error) {
	groupMembers := NewStashGroupMembers(c)
	opts := &ListOptions{}
	apiObjs := []*User{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /groups
		paging, listErr := groupMembers.List(ctx, groupID.(string), opts)
		apiObjs = append(apiObjs, groupMembers.getGroupMembers()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateUserAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) GetProject(ctx context.Context, projectName string) (*Project, error) {
	projects := NewStashProjects(c)
	project, err := projects.Get(ctx, projectName)
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	if err := validateProjectAPI(project); err != nil {
		return nil, err
	}

	return project, nil
}

func (c *stashClientImpl) ListProjects(ctx context.Context) ([]*Project, error) {
	projects := NewStashProjects(c)
	apiObjs := []*Project{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := projects.List(ctx, opts)
		apiObjs = append(apiObjs, projects.getProjects()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateProjectAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) ListProjectGroups(ctx context.Context, projectName string) ([]*ProjectGroupPermission, error) {
	projectGroups := NewStashProjectGroups(c, c.getOwnerID(ctx, projectName))
	apiObjs := []*ProjectGroupPermission{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := projectGroups.List(ctx, opts)
		apiObjs = append(apiObjs, projectGroups.getGroups()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateProjectGroupPermissionAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) ListRepositoryTeamPermissions(ctx context.Context, projectName, repoName string) ([]*RepositoryGroupPermission, error) {
	repoTeams := NewStashRepositoryGroups(c, c.getOwnerID(ctx, projectName), repoName)
	apiObjs := []*RepositoryGroupPermission{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := repoTeams.List(ctx, opts)
		apiObjs = append(apiObjs, repoTeams.getGroups()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateRepositoryGroupPermissionAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) GetRepositoryTeamPermissions(ctx context.Context, orgName, repo, teamName string) (string, error) {
	repositoryPermissions := NewStashRepositoryGroups(c, c.getOwnerID(ctx, orgName), repo)
	permission, err := repositoryPermissions.Get(ctx, teamName)
	if err != nil {
		return "", err
	}
	// Validate the API objects
	if err := validateRepositoryGroupPermissionAPI(permission); err != nil {
		return "", err
	}

	return permission.Permission, nil
}

func (c *stashClientImpl) CreateRepositoryTeamPermissions(ctx context.Context, orgName, repo, teamName, permission string) error {
	repositoryPermissions := NewStashRepositoryGroups(c, c.getOwnerID(ctx, orgName), repo)
	perm := &RepositoryGroupPermission{}
	perm.Group.Name = teamName
	perm.Permission = permission

	// Validate the API objects
	if err := validateRepositoryGroupPermissionAPI(perm); err != nil {
		return err
	}

	return repositoryPermissions.Create(ctx, perm)
}

func (c *stashClientImpl) GetRepository(ctx context.Context, owner, name string) (*Repository, error) {
	repositories := NewStashRepositories(c)
	repository, err := repositories.Get(ctx, c.getOwnerID(ctx, owner), name)
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	if err := validateRepositoryAPI(repository); err != nil {
		return nil, err
	}

	return repository, nil
}

func (c *stashClientImpl) ListRepositories(ctx context.Context, owner string) ([]*Repository, error) {
	repositories := NewStashRepositories(c)
	apiObjs := []*Repository{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := repositories.List(ctx, c.getOwnerID(ctx, owner), opts)
		apiObjs = append(apiObjs, repositories.getRepositories()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateRepositoryAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) CreateRepository(ctx context.Context, owner string, req *Repository) (*Repository, error) {
	// First thing, validate and default the request to ensure a valid and fully-populated object
	// (to minimize any possible diffs between desired and actual state)
	if err := gitprovider.ValidateAndDefaultInfo(req); err != nil {
		return nil, err
	}
	c.log.V(1).Info("create repository", "org", owner, "name", req.Name)
	repositories := NewStashRepositories(c)
	apiObj, err := repositories.Create(ctx, c.getOwnerID(ctx, owner), req)
	if err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *stashClientImpl) UpdateRepository(ctx context.Context, owner string, req *Repository) (*Repository, error) {
	repositories := NewStashRepositories(c)
	apiObj, err := repositories.Update(ctx, c.getOwnerID(ctx, owner), req)
	if err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *stashClientImpl) DeleteRepository(ctx context.Context, owner, name string) error {
	repositories := NewStashRepositories(c)
	err := repositories.Delete(ctx, c.getOwnerID(ctx, owner), name)
	if err != nil {
		return err
	}
	return nil
}

func (c *stashClientImpl) ListKeys(ctx context.Context, owner, repoName string) ([]*DeployKey, error) {
	deployKeys := NewStashDeployKeys(c)
	apiObjs := []*DeployKey{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := deployKeys.List(ctx, c.getOwnerID(ctx, owner), repoName, opts)
		apiObjs = append(apiObjs, deployKeys.getDeployKeys()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validateDeployKeyAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) CreateKey(ctx context.Context, req *DeployKey) (*DeployKey, error) {
	deployKeys := NewStashDeployKeys(c)
	req.Repository.Project.Key = c.getOwnerID(ctx, req.Repository.Project.Name)
	apiObj, err := deployKeys.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *stashClientImpl) DeleteKey(ctx context.Context, owner, repoName string, keyID int) error {
	return NewStashDeployKeys(c).Delete(ctx, c.getOwnerID(ctx, owner), repoName, keyID)
}

func (c *stashClientImpl) UpdateKey(ctx context.Context, owner, repoName string, keyID int, permission string) (*DeployKey, error) {
	deployKeys := NewStashDeployKeys(c)
	apiObj, err := deployKeys.Update(ctx, c.getOwnerID(ctx, owner), repoName, keyID, permission)
	if err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *stashClientImpl) ListPullRequests(ctx context.Context, owner, repoName string) ([]*PullRequest, error) {
	pullRequests := NewStashPullRequests(c)
	apiObjs := []*PullRequest{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := pullRequests.List(ctx, c.getOwnerID(ctx, owner), repoName, opts)
		apiObjs = append(apiObjs, pullRequests.getPullRequests()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	// Validate the API objects
	for _, apiObj := range apiObjs {
		if err := validatePullRequestsAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *stashClientImpl) GetPullRequest(ctx context.Context, owner, repoName string, prID int) (*PullRequest, error) {
	pullRequests := NewStashPullRequests(c)
	apiObj, err := pullRequests.Get(ctx, c.getOwnerID(ctx, owner), repoName, prID)
	if err != nil {
		return nil, err
	}
	if err := validatePullRequestsAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *stashClientImpl) CreatePullRequest(ctx context.Context, owner, repoName string, pr *PullRequestCreation) (*PullRequest, error) {
	pullRequests := NewStashPullRequests(c)
	apiObj, err := pullRequests.Create(ctx, c.getOwnerID(ctx, owner), repoName, pr)
	if err != nil {
		return nil, err
	}
	if err := validatePullRequestsAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *stashClientImpl) DeletePullRequest(ctx context.Context, owner, repoName string, prID int) error {
	pullRequests := NewStashPullRequests(c)
	return pullRequests.Delete(ctx, c.getOwnerID(ctx, owner), repoName, prID)
}

func (c *stashClientImpl) UpdatePullRequest(ctx context.Context, owner, repoName string, pr *PullRequest) (*PullRequest, error) {
	pullRequests := NewStashPullRequests(c)
	apiObj, err := pullRequests.Update(ctx, c.getOwnerID(ctx, pr.ToRef.Repository.Project.Name), pr.ToRef.Repository.Name, pr)
	if err != nil {
		return nil, err
	}
	if err := validatePullRequestsAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *stashClientImpl) ListCommitsPage(ctx context.Context, owner, repoName, branch string, perPage int, page int) ([]*Commit, error) {
	commits := NewStashCommits(c)
	apiObjs := []*Commit{}
	opts := &ListOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := commits.List(ctx, c.getOwnerID(ctx, owner), repoName, opts)
		apiObjs = append(apiObjs, commits.getCommits()...)
		return paging, listErr
	})
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

func (c *stashClientImpl) CreateBranch(ctx context.Context, owner, repoName, branch string) error {
	return nil
}

func (c *stashClientImpl) GetCommit(ctx context.Context, owner, repoName, commitID string) (*Commit, error) {
	commits := NewStashCommits(c)
	apiObjs, err := commits.Get(ctx, c.getOwnerID(ctx, owner), repoName, commitID)
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}
