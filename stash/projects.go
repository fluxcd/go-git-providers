/*
Copyright 2021 The Flux authors

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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	projectsURI        = "projects"
	groupPermisionsURI = "permissions/groups"
	userPermisionsURI  = "permissions/users"
)

// Projects interface defines the methods that can be used to
// retrieve projects and related permissions.
type Projects interface {
	List(ctx context.Context, opts *PagingOptions) (*ProjectsList, error)
	Get(ctx context.Context, projectName string) (*Project, error)
	All(ctx context.Context) ([]*Project, error)
	GetProjectGroupPermission(ctx context.Context, projectKey, groupName string) (*ProjectGroupPermission, error)
	ListProjectGroupsPermission(ctx context.Context, projectKey string, opts *PagingOptions) (*ProjectGroups, error)
	AllGroupsPermission(ctx context.Context, projectKey string) ([]*ProjectGroupPermission, error)
	ListProjectUsersPermission(ctx context.Context, projectKey string, opts *PagingOptions) (*ProjectUsers, error)
}

// ProjectsService is a client for communicating with stash projects endpoint
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
type ProjectsService service

// Project represents a Stash project
// which is a way for teams to group, manage, and organize their repositories.
type Project struct {
	// Session is the http.Response of the last request made to the Stash API.
	Session `json:"sessionInfo,omitempty"`
	// Description is the project description.
	Description string `json:"description,omitempty"`
	// ID is the project ID.
	ID int64 `json:"id,omitempty"`
	// Key is the project key.
	Key string `json:"key,omitempty"`
	// Links is the project hyperlinks.
	Links `json:"links,omitempty"`
	// User is the the authenticated user.
	User `json:"owner,omitempty"`
	// Name is the project name.
	Name string `json:"name,omitempty"`
	// Public is the project public flag.
	Public bool `json:"public,omitempty"`
	// Type is the project type.
	Type string `json:"type,omitempty"`
}

// ProjectsList is a list of projects
type ProjectsList struct {
	// Paging is the paging information.
	Paging
	// Projects is the list of projects.
	Projects []*Project `json:"values,omitempty"`
}

// GetProjects returns a slice of Project.
func (p *ProjectsList) GetProjects() []*Project {
	return p.Projects
}

// List retrieves a list of projects.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a ProjectsList struct is returned. It contains paging information to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/projects".
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *ProjectsService) List(ctx context.Context, opts *PagingOptions) (*ProjectsList, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("get projects request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list projects failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("list projects failed: %s", resp.Status)
	}

	p := &ProjectsList{
		Projects: []*Project{},
	}
	if err := json.Unmarshal(res, p); err != nil {
		return nil, fmt.Errorf("list projects failed, unable to unmarshal repository list json: %w", err)
	}

	for _, r := range p.GetProjects() {
		r.Session.set(resp)
	}

	return p, nil
}

// All retrieves all projects.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *ProjectsService) All(ctx context.Context) ([]*Project, error) {
	p := []*Project{}
	opts := &PagingOptions{Limit: perPageLimit}
	err := allPages(opts, func() (*Paging, error) {
		list, err := s.List(ctx, opts)
		if err != nil {
			return nil, err
		}
		p = append(p, list.GetProjects()...)
		return &list.Paging, nil
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Get retrieves a project by Name.
// Get uses the endpoint "GET /rest/api/1.0/projects/?name&permission".
// The authenticated user must have PROJECT_VIEW permission for the specified project to call this resource.
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *ProjectsService) Get(ctx context.Context, projectName string) (*Project, error) {
	query := url.Values{
		"name": []string{projectName},
	}
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("get project request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get project failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	p := &ProjectsList{
		Projects: []*Project{},
	}
	if err := json.Unmarshal(res, &p); err != nil {
		return nil, fmt.Errorf("get project failed, unable to unmarshal repository list json: %w", err)
	}

	if len(p.Projects) == 0 {
		return nil, ErrNotFound
	}

	// find the project
	for i := range p.Projects {
		if p.Projects[i].Name == projectName {
			// Found!
			p.Projects[i].Session.set(resp)
			return p.Projects[i], nil
		}
	}

	return nil, ErrNotFound

}

// ProjectGroupPermission is a permission for a given group.
// The permission is tied to a project.
// The permission can be either read, write, or admin.
type ProjectGroupPermission struct {
	// Session is the http.Response of the last request made to the Stash API.
	Session Session `json:"sessionInfo,omitempty"`
	// Group is the group that the permission is for.
	Group struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	// Permission denotes a group's permission level. Available project permissions are:
	// PROJECT_READ
	// PROJECT_WRITE
	// PROJECT_ADMIN
	Permission string `json:"permission,omitempty"`
}

// ProjectGroups represents a list of groups for a given project.
type ProjectGroups struct {
	// Paging is the paging information.
	Paging
	// ProjectKey is the Key of the project.
	ProjectKey string `json:"-"`
	// Groups is the list of groups permissions.
	Groups []*ProjectGroupPermission `json:"values,omitempty"`
}

// GetGroups returns a slice of ProjectGroupPermission.
func (p *ProjectGroups) GetGroups() []*ProjectGroupPermission {
	return p.Groups
}

// GetProjectGroupPermission retrieve a group that have been granted at least one permission for the specified project.
// GetRepositoryGroupPermission uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/permissions/groups?filter".
// The authenticated user must have PROJECT_ADMIN permission for the specified project
func (s *ProjectsService) GetProjectGroupPermission(ctx context.Context, projectKey, groupName string) (*ProjectGroupPermission, error) {
	query := url.Values{
		filterKey: []string{groupName},
	}
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, groupPermisionsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("get group permissions request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get group permissions to project failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	permissions := &ProjectGroups{}
	if err := json.Unmarshal(res, permissions); err != nil {
		return nil, fmt.Errorf("get group permissions for project failed, unable to unmarshall project group json: %w", err)
	}

	if len(permissions.Groups) == 0 {
		return nil, ErrNotFound
	}

	permissions.Groups[0].Session.set(resp)
	return permissions.Groups[0], nil
}

// ListProjectGroupsPermission retrieves a list of groups and their permissions for a given project.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a ProjectGroups struct is returned. It contains paging information to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/permissions/groups".
// The authenticated user must have PROJECT_ADMIN permission for the specified project
// or a higher global permission to call this resource.
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *ProjectsService) ListProjectGroupsPermission(ctx context.Context, projectKey string, opts *PagingOptions) (*ProjectGroups, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, groupPermisionsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("get project groups permission request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list project groups permission failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	gp := &ProjectGroups{
		ProjectKey: projectKey,
		Groups:     []*ProjectGroupPermission{},
	}
	if err := json.Unmarshal(res, gp); err != nil {
		return nil, fmt.Errorf("list project groups permission failed, unable to unmarshal project groups json: %w", err)
	}

	for _, r := range gp.GetGroups() {
		r.Session.set(resp)
	}

	return gp, nil
}

// AllGroupsPermission retrieves all projects groups permission.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *ProjectsService) AllGroupsPermission(ctx context.Context, projectKey string) ([]*ProjectGroupPermission, error) {
	p := []*ProjectGroupPermission{}
	opts := &PagingOptions{Limit: perPageLimit}
	err := allPages(opts, func() (*Paging, error) {
		list, err := s.ListProjectGroupsPermission(ctx, projectKey, opts)
		if err != nil {
			return nil, err
		}
		p = append(p, list.GetGroups()...)
		return &list.Paging, nil
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}

// ProjectUserPermission is a permission for a given User.
// The permission is tied to a project.
// The permission can be either read, write, or admin.
type ProjectUserPermission struct {
	// Session is the http.Response of the last request made to the Stash API.
	Session Session `json:"sessionInfo,omitempty"`
	// User is the user that the permission is for.
	User User `json:"user,omitempty"`
	// Permission denotes a group's permission level. Available project permissions are:
	// PROJECT_READ
	// PROJECT_WRITE
	// PROJECT_ADMIN
	Permission string `json:"permission,omitempty"`
}

// ProjectUsers represents a list of users for a given project.
type ProjectUsers struct {
	// Paging is the paging information.
	Paging
	// ProjectKey is the key of the project.
	ProjectKey string `json:"-"`
	// Users is the list of users permissions.
	Users []*ProjectUserPermission `json:"values,omitempty"`
}

// GetUsers returns a slice of ProjectUserPermission.
func (p *ProjectUsers) GetUsers() []*ProjectUserPermission {
	return p.Users
}

// ListProjectUsersPermission retrieves a list of users and their permissions for a given project.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a ProjectUsers struct is returned. It contains paging information to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/permissions/users".
// The authenticated user must have PROJECT_ADMIN permission for the specified project
// or a higher global permission to call this resource.
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *ProjectsService) ListProjectUsersPermission(ctx context.Context, projectKey string, opts *PagingOptions) (*ProjectUsers, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, userPermisionsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("get project users permission request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list project users permission failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	up := &ProjectUsers{
		ProjectKey: projectKey,
		Users:      []*ProjectUserPermission{},
	}
	if err := json.Unmarshal(res, up); err != nil {
		return nil, fmt.Errorf("list project users permission failed, unable to unmarshal repository list json: %w", err)
	}

	for _, r := range up.GetUsers() {
		r.Session.set(resp)
	}

	return up, nil
}
