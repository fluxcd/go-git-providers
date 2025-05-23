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

package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"gitlab.com/gitlab-org/api/client-go"
)

// gitlabClientImpl is a wrapper around *gitlab.Client, which implements higher-level methods,
// operating on the go-gitlab structs. Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
// This interface is also fakeable, in order to unit-test the client.
type gitlabClient interface {
	// Client returns the underlying *gitlab.Client
	Client() *gitlab.Client

	// Group methods

	// GetGroup is a wrapper for "GET /groups/{group}".
	// This function HTTP error wrapping, and validates the server result.
	GetGroup(ctx context.Context, groupID interface{}) (*gitlab.Group, error)
	// ListGroups is a wrapper for "GET /groups".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListGroups(ctx context.Context) ([]*gitlab.Group, error)
	// ListSubgroups is a wrapper for "GET /groups/{group}/subgroups".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListSubgroups(ctx context.Context, groupName string) ([]*gitlab.Group, error)
	// ListGroupMembers is a wrapper for "GET /groups/{group}/members".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListGroupMembers(ctx context.Context, groupName string) ([]*gitlab.GroupMember, error)

	// Project methods

	// GetProject is a wrapper for "GET /projects/{project}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetGroupProject(ctx context.Context, groupName string, projectName string) (*gitlab.Project, error)
	// ListGroupProjects is a wrapper for "GET /groups/{group}/projects".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListGroupProjects(ctx context.Context, groupName string) ([]*gitlab.Project, error)
	// GetProject is a wrapper for "GET /projects/{project}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetUserProject(ctx context.Context, projectName string) (*gitlab.Project, error)
	// ListUserProjects is a wrapper for "GET /users/{username}/projects".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListUserProjects(ctx context.Context, username string) ([]*gitlab.Project, error)
	// ListProjectUsers is a wrapper for "GET /projects/{project}/users".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListProjectUsers(ctx context.Context, projectName string) ([]*gitlab.ProjectUser, error)
	// CreateProject is a wrapper for "POST /projects"
	// This function handles HTTP error wrapping, and validates the server result.
	CreateProject(ctx context.Context, req *gitlab.Project, opts *gitlab.CreateProjectOptions) (*gitlab.Project, error)
	// UpdateProject is a wrapper for "PUT /projects/{project}".
	// This function handles HTTP error wrapping, and validates the server result.
	UpdateProject(ctx context.Context, req *gitlab.Project) (*gitlab.Project, error)
	// DeleteProject is a wrapper for "DELETE /projects/{project}".
	// This function handles HTTP error wrapping.
	// DANGEROUS COMMAND: In order to use this, you must set destructiveActions to true.
	DeleteProject(ctx context.Context, projectName string) error

	// GetUser is a wrapper for "GET /user"
	GetUser(ctx context.Context) (*gitlab.User, error)

	// Deploy key methods

	// ListKeys is a wrapper for "GET /projects/{project}/deploy_keys".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListKeys(projectName string) ([]*gitlab.ProjectDeployKey, error)
	// CreateProjectKey is a wrapper for "POST /projects/{project}/deploy_keys".
	// This function handles HTTP error wrapping, and validates the server result.
	CreateKey(projectName string, req *gitlab.ProjectDeployKey) (*gitlab.ProjectDeployKey, error)
	// DeleteKey is a wrapper for "DELETE /projects/{project}/deploy_keys/{key_id}".
	// This function handles HTTP error wrapping.
	DeleteKey(projectName string, keyID int) error

	// Deploy token methods

	// ListTokens is a wrapper for "GET /projects/{project}/deploy_tokens".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListTokens(projectName string) ([]*gitlab.DeployToken, error)
	// CreateProjectKey is a wrapper for "POST /projects/{project}/deploy_tokens".
	// This function handles HTTP error wrapping, and validates the server result.
	CreateToken(projectName string, req *gitlab.DeployToken) (*gitlab.DeployToken, error)
	// DeleteKey is a wrapper for "DELETE /projects/{project}/deploy_tokens/{key_id}".
	// This function handles HTTP error wrapping.
	DeleteToken(projectName string, keyID int) error

	// Team related methods

	// ShareGroup is a wrapper for ""
	// This function handles HTTP error wrapping, and validates the server result.
	ShareProject(projectName string, groupID, groupAccess int) error
	// UnshareProject is a wrapper for ""
	// This function handles HTTP error wrapping, and validates the server result.
	UnshareProject(projectName string, groupID int) error

	// Commits

	// ListCommitsPage is a wrapper for "GET /projects/{project}/repository/commits".
	// This function handles pagination, HTTP error wrapping.
	ListCommitsPage(projectName, branch string, perPage int, page int) ([]*gitlab.Commit, error)
}

// gitlabClientImpl is a wrapper around *gitlab.Client, which implements higher-level methods,
// operating on the go-gitlab structs. See the gitlabClient interface for method documentation.
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
type gitlabClientImpl struct {
	c                  *gitlab.Client
	destructiveActions bool
}

// gitlabClientImpl implements gitlabClient.
var _ gitlabClient = &gitlabClientImpl{}

func (c *gitlabClientImpl) Client() *gitlab.Client {
	return c.c
}

func (c *gitlabClientImpl) GetGroup(ctx context.Context, groupID interface{}) (*gitlab.Group, error) {
	apiObj, _, err := c.c.Groups.GetGroup(groupID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	// Validate the API object
	if err := validateGroupAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *gitlabClientImpl) ListGroups(ctx context.Context) ([]*gitlab.Group, error) {
	apiObjs := []*gitlab.Group{}
	opts := &gitlab.ListGroupsOptions{}
	err := allGroupPages(opts, func() (*gitlab.Response, error) {
		// GET /groups
		pageObjs, resp, listErr := c.c.Groups.ListGroups(opts, gitlab.WithContext(ctx))
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
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

func (c *gitlabClientImpl) ListSubgroups(ctx context.Context, groupName string) ([]*gitlab.Group, error) {
	var apiObjs []*gitlab.Group
	opts := &gitlab.ListSubGroupsOptions{}
	err := allSubgroupPages(opts, func() (*gitlab.Response, error) {
		// GET /groups
		pageObjs, resp, listErr := c.c.Groups.ListSubGroups(groupName, opts, gitlab.WithContext(ctx))
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
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

func (c *gitlabClientImpl) GetGroupProject(ctx context.Context, groupName string, projectName string) (*gitlab.Project, error) {
	opts := &gitlab.GetProjectOptions{}
	apiObj, _, err := c.c.Projects.GetProject(fmt.Sprintf("%s/%s", strings.ToLower(groupName), projectName), opts, gitlab.WithContext(ctx))
	return validateProjectAPIResp(apiObj, err)
}

func (c *gitlabClientImpl) ListGroupProjects(ctx context.Context, groupName string) ([]*gitlab.Project, error) {
	var apiObjs []*gitlab.Project
	opts := &gitlab.ListGroupProjectsOptions{}
	err := allGroupProjectPages(opts, func() (*gitlab.Response, error) {
		pageObjs, resp, listErr := c.c.Groups.ListGroupProjects(groupName, opts, gitlab.WithContext(ctx))
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}
	return validateProjectObjects(apiObjs)
}

func validateProjectObjects(apiObjs []*gitlab.Project) ([]*gitlab.Project, error) {
	for _, apiObj := range apiObjs {
		// Make sure apiObj is valid
		if err := validateProjectAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *gitlabClientImpl) ListGroupMembers(ctx context.Context, groupName string) ([]*gitlab.GroupMember, error) {
	var apiObjs []*gitlab.GroupMember
	opts := &gitlab.ListGroupMembersOptions{}
	err := allGroupMemberPages(opts, func() (*gitlab.Response, error) {
		// GET /groups/{group}/members
		pageObjs, resp, listErr := c.c.Groups.ListGroupMembers(groupName, opts, gitlab.WithContext(ctx))
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

func (c *gitlabClientImpl) GetUserProject(ctx context.Context, projectName string) (*gitlab.Project, error) {
	opts := &gitlab.GetProjectOptions{}
	apiObj, _, err := c.c.Projects.GetProject(projectName, opts, gitlab.WithContext(ctx))
	return validateProjectAPIResp(apiObj, err)
}

func validateProjectAPIResp(apiObj *gitlab.Project, err error) (*gitlab.Project, error) {
	// If the response contained an error, return
	if err != nil {
		return nil, handleHTTPError(err)
	}
	// Make sure apiObj is valid
	if err := validateProjectAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *gitlabClientImpl) ListProjects(ctx context.Context) ([]*gitlab.Project, error) {
	var apiObjs []*gitlab.Project
	opts := &gitlab.ListProjectsOptions{}
	err := allProjectPages(opts, func() (*gitlab.Response, error) {
		// GET /projects
		pageObjs, resp, listErr := c.c.Projects.ListProjects(opts, gitlab.WithContext(ctx))
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

func (c *gitlabClientImpl) ListProjectUsers(ctx context.Context, projectName string) ([]*gitlab.ProjectUser, error) {
	var apiObjs []*gitlab.ProjectUser
	opts := &gitlab.ListProjectUserOptions{}
	err := allProjectUserPages(opts, func() (*gitlab.Response, error) {
		// GET /projects/{project}/users
		pageObjs, resp, listErr := c.c.Projects.ListProjectsUsers(projectName, opts, gitlab.WithContext(ctx))
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

func (c *gitlabClientImpl) ListUserProjects(ctx context.Context, username string) ([]*gitlab.Project, error) {
	var apiObjs []*gitlab.Project
	opts := &gitlab.ListProjectsOptions{}
	err := allProjectPages(opts, func() (*gitlab.Response, error) {
		// GET /projects/{project}/users
		pageObjs, resp, listErr := c.c.Projects.ListUserProjects(username, opts, gitlab.WithContext(ctx))
		apiObjs = append(apiObjs, pageObjs...)
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}
	return apiObjs, nil
}

func (c *gitlabClientImpl) CreateProject(ctx context.Context, req *gitlab.Project, extraOpts *gitlab.CreateProjectOptions) (*gitlab.Project, error) {
	var namespaceID int
	// If the project doesn't belong to a user set its namespace ID
	if req.Namespace != nil && req.Namespace.Kind != "user" {
		group, err := c.GetGroup(ctx, req.Namespace.Name)
		if err != nil {
			return nil, err
		}
		namespaceID = group.ID
	}

	opts := extraOpts
	if opts == nil {
		opts = &gitlab.CreateProjectOptions{}
	}
	opts.Name = &req.Name
	opts.DefaultBranch = &req.DefaultBranch
	opts.Description = &req.Description
	opts.Visibility = &req.Visibility
	if namespaceID != 0 {
		opts.NamespaceID = &namespaceID
	}

	apiObj, _, err := c.c.Projects.CreateProject(opts, gitlab.WithContext(ctx))
	return validateProjectAPIResp(apiObj, err)
}

func (c *gitlabClientImpl) UpdateProject(ctx context.Context, req *gitlab.Project) (*gitlab.Project, error) {
	opts := &gitlab.EditProjectOptions{
		Name:        &req.Name,
		Description: &req.Description,
		Visibility:  &req.Visibility,
	}
	apiObj, _, err := c.c.Projects.EditProject(req.ID, opts, gitlab.WithContext(ctx))
	return validateProjectAPIResp(apiObj, err)
}

func (c *gitlabClientImpl) DeleteProject(ctx context.Context, projectName string) error {
	// Don't allow deleting repositories if the user didn't explicitly allow dangerous API calls.
	if !c.destructiveActions {
		return fmt.Errorf("cannot delete repository: %w", gitprovider.ErrDestructiveCallDisallowed)
	}
	// DELETE /projects/{project}
	_, err := c.c.Projects.DeleteProject(projectName, nil)
	return err
}

func (c *gitlabClientImpl) GetUser(ctx context.Context) (*gitlab.User, error) {
	// GET /user
	proj, _, err := c.c.Users.CurrentUser(gitlab.WithContext(ctx))
	return proj, err
}

func (c *gitlabClientImpl) ListKeys(projectName string) ([]*gitlab.ProjectDeployKey, error) {
	apiObjs := []*gitlab.ProjectDeployKey{}
	opts := &gitlab.ListProjectDeployKeysOptions{}
	err := allDeployKeyPages(opts, func() (*gitlab.Response, error) {
		// GET /projects/{project}/deploy_keys
		pageObjs, resp, listErr := c.c.DeployKeys.ListProjectDeployKeys(projectName, opts)
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

func (c *gitlabClientImpl) CreateKey(projectName string, req *gitlab.ProjectDeployKey) (*gitlab.ProjectDeployKey, error) {
	opts := &gitlab.AddDeployKeyOptions{
		Title:   &req.Title,
		Key:     &req.Key,
		CanPush: &req.CanPush,
	}
	// POST /projects/{project}/deploy_keys
	apiObj, _, err := c.c.DeployKeys.AddDeployKey(projectName, opts)
	if err != nil {
		return nil, handleHTTPError(err)
	}
	if err := validateDeployKeyAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *gitlabClientImpl) DeleteKey(projectName string, keyID int) error {
	// DELETE /projects/{project}/deploy_keys
	_, err := c.c.DeployKeys.DeleteDeployKey(projectName, keyID)
	return handleHTTPError(err)
}

func (c *gitlabClientImpl) ListTokens(projectName string) ([]*gitlab.DeployToken, error) {
	apiObjs := []*gitlab.DeployToken{}
	opts := &gitlab.ListProjectDeployTokensOptions{}
	err := allDeployTokenPages(opts, func() (*gitlab.Response, error) {
		// GET /projects/{project}/deploy_tokens
		pageObjs, resp, listErr := c.c.DeployTokens.ListProjectDeployTokens(projectName, opts)
		// filter for active tokens
		for _, apiObj := range pageObjs {
			if !apiObj.Expired && !apiObj.Revoked {
				apiObjs = append(apiObjs, apiObj)
			}
		}
		return resp, listErr
	})
	if err != nil {
		return nil, err
	}

	for _, apiObj := range apiObjs {
		if err := validateDeployTokenAPI(apiObj); err != nil {
			return nil, err
		}
	}
	return apiObjs, nil
}

func (c *gitlabClientImpl) CreateToken(projectName string, req *gitlab.DeployToken) (*gitlab.DeployToken, error) {
	opts := &gitlab.CreateProjectDeployTokenOptions{
		Name:   &req.Name,
		Scopes: &[]string{"read_repository"},
	}
	// POST /projects/{project}/deploy_tokens
	apiObj, _, err := c.c.DeployTokens.CreateProjectDeployToken(projectName, opts)
	if err != nil {
		return nil, handleHTTPError(err)
	}
	if err := validateDeployTokenAPI(apiObj); err != nil {
		return nil, err
	}
	return apiObj, nil
}

func (c *gitlabClientImpl) DeleteToken(projectName string, keyID int) error {
	// DELETE /projects/{project}/deploy_tokens/{deploy_token_id}
	_, err := c.c.DeployTokens.DeleteProjectDeployToken(projectName, keyID)
	return handleHTTPError(err)
}

func (c *gitlabClientImpl) ShareProject(projectName string, groupIDObj, groupAccessObj int) error {
	groupAccess := gitlab.AccessLevel(gitlab.AccessLevelValue(groupAccessObj))
	groupID := &groupIDObj
	opt := &gitlab.ShareWithGroupOptions{
		GroupID:     groupID,
		GroupAccess: groupAccess,
	}

	_, err := c.c.Projects.ShareProjectWithGroup(projectName, opt)
	return handleHTTPError(err)
}

func (c *gitlabClientImpl) UnshareProject(projectName string, groupID int) error {
	_, err := c.c.Projects.DeleteSharedProjectFromGroup(projectName, groupID)
	return handleHTTPError(err)
}

func (c *gitlabClientImpl) ListCommitsPage(projectName string, branch string, perPage int, page int) ([]*gitlab.Commit, error) {
	apiObjs := make([]*gitlab.Commit, 0)

	opts := gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
		RefName: &branch,
	}

	// GET /projects/{id}/repository/commits
	pageObjs, _, listErr := c.c.Commits.ListCommits(projectName, &opts)
	for _, c := range pageObjs {
		apiObjs = append(apiObjs, &gitlab.Commit{
			ID:         c.ID,
			AuthorName: c.AuthorName,
			Message:    c.Message,
			CreatedAt:  c.CreatedAt,
			WebURL:     c.WebURL,
		})
	}

	if listErr != nil {
		return nil, listErr
	}
	return apiObjs, nil
}
