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

	gostash "github.com/fluxcd/go-git-providers/go-stash"
)

// stashClientImpl is a wrapper around http.Client, which implements rest API access,
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
// This interface is also fakeable, in order to unit-test the client.
type stashClient interface {
	// Client returns the underlying http.Client
	Client() *gostash.Client

	// Group methods

	// GetGroup is a wrapper for "GET /rest/api/1.0/admin/groups?filter={group}".
	// This function HTTP error wrapping, and validates the server result.
	GetGroup(ctx context.Context, groupID interface{}) (*gostash.Group, error)

	// ListGroups is a wrapper for "GET /rest/api/1.0/admin/groups".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListGroups(ctx context.Context) ([]*gostash.Group, error)

	// ListGroupMembers is a wrapper for "GET /rest/api/1.0/admin/groups/more-members?context={group}".
	// It retruns the users who are members of a group/project
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListGroupMembers(ctx context.Context, groupID interface{}) ([]*gostash.User, error)

	// Project methods

	// GetProject is a wrapper for "GET /rest/api/1.0/projects?filter={project}".
	// This function handles HTTP error wrapping, and validates the server result.
	GetProject(ctx context.Context, projectName string) (*gostash.Project, error)

	// ListProjects is a wrapper for "GET /rest/api/1.0/projects".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListProjects(ctx context.Context) ([]*gostash.Project, error)

	// ListProjectGroups is a wrapper for "GET /rest/api/1.0/projects/permissions/groups".
	// This function handles pagination, HTTP error wrapping, and validates the server result.
	ListProjectGroups(ctx context.Context, projectName string) ([]*gostash.ProjectGroupPermission, error)
}

// stashClientImpl is a wrapper around http.Client that implements the StashClient interface.
// Pagination is implemented for all List* methods, all returned
// objects are validated, and HTTP errors are handled/wrapped using handleHTTPError.
type stashClientImpl struct {
	c                  *gostash.Client
	destructiveActions bool
	log                logr.Logger
}

// stashClientImpl implements stashClient.
var _ stashClient = &stashClientImpl{}

// Client returns the underlying http.Client
func (c *stashClientImpl) Client() *gostash.Client {
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
	// TO DO : handle error handleHTTPError
	if err != nil {
		return addTilde(projectName)
	}
	return project.Key
}

// GetGroup is a wrapper for "GET /rest/api/1.0/admin/groups?filter={group}".
// This function HTTP error wrapping, and validates the server result.
func (c *stashClientImpl) GetGroup(ctx context.Context, groupID interface{}) (*gostash.Group, error) {
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

// ListGroups is a wrapper for "GET /rest/api/1.0/admin/groups".
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (c *stashClientImpl) ListGroups(ctx context.Context) ([]*gostash.Group, error) {
	groups := NewStashGroups(c)
	apiObjs := []*gostash.Group{}
	opts := &gostash.PagingOptions{}
	err := allPages(opts, func() (*gostash.Paging, error) {
		// GET /groups
		paging, listErr := groups.List(ctx, opts)
		apiObjs = append(apiObjs, groups.GetGroups()...)
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

// ListGroupMembers is a wrapper for "GET /rest/api/1.0/admin/groups/more-members?context={group}".
// It retruns the users who are members of a group/project
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (c *stashClientImpl) ListGroupMembers(ctx context.Context, groupID interface{}) ([]*User, error) {
	groupMembers := NewStashGroupMembers(c)
	opts := &gostash.PagingOptions{}
	apiObjs := []*User{}
	err := allPages(opts, func() (*Paging, error) {
		// GET group members
		paging, listErr := groupMembers.List(ctx, groupID.(string), opts)
		apiObjs = append(apiObjs, groupMembers.GetGroupMembers()...)
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

// GetProject is a wrapper for "GET /rest/api/1.0/projects?filter={project}".
// This function handles HTTP error wrapping, and validates the server result.
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

// ListProjects is a wrapper for "GET /rest/api/1.0/projects".
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (c *stashClientImpl) ListProjects(ctx context.Context) ([]*Project, error) {
	projects := NewStashProjects(c)
	apiObjs := []*Project{}
	opts := &PagingOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := projects.List(ctx, opts)
		apiObjs = append(apiObjs, projects.GetProjects()...)
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

// ListProjectGroups is a wrapper for "GET /rest/api/1.0/projects/permissions/groups".
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (c *stashClientImpl) ListProjectGroups(ctx context.Context, projectName string) ([]*ProjectGroupPermission, error) {
	projectGroups := NewStashProjectGroups(c, c.getOwnerID(ctx, projectName))
	apiObjs := []*ProjectGroupPermission{}
	opts := &PagingOptions{}
	err := allPages(opts, func() (*Paging, error) {
		// GET /projects
		paging, listErr := projectGroups.List(ctx, opts)
		apiObjs = append(apiObjs, projectGroups.GetGroups()...)
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
