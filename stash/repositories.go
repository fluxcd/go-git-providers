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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var (
	// ErrorGetRepositoryMultipleItems is returned when the response contains more than one item.
	ErrorGetRepositoryMultipleItems = errors.New("multiple items returned for repo name")
	// ErrAlreadyExists is returned when the repository already exists.
	ErrAlreadyExists = errors.New("resource already exists, cannot create object")
)

const (
	// RepositoriesURI is the URI for the repositories endpoint
	RepositoriesURI = "repos"
)

// Repositories interface defines the operations for working with repositories.
type Repositories interface {
	RepositoryManager
	RepositoryPermissionManager
}

// RepositoryManager interface defines the CRUD operations for repositories.
type RepositoryManager interface {
	List(ctx context.Context, projectKey string, opts *PagingOptions) (*RepositoryList, error)
	All(ctx context.Context, projectKey string, maxPages int) ([]*Repository, error)
	Get(ctx context.Context, projectKey, repoSlug string) (*Repository, error)
	Create(ctx context.Context, projectKey string, repository *Repository) (*Repository, error)
	Update(ctx context.Context, projectKey, repositorySlug string, repository *Repository) (*Repository, error)
	Delete(ctx context.Context, projectKey, repoSlug string) error
}

// RepositoryPermissionManager interface defines the operations for working with repository permissions.
type RepositoryPermissionManager interface {
	GetRepositoryGroupPermission(ctx context.Context, projectKey, repositorySlug, groupName string) (*RepositoryGroupPermission, error)
	ListRepositoryGroupsPermission(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*RepositoryGroups, error)
	AllGroupsPermission(ctx context.Context, projectKey, repositorySlug string, maxPages int) ([]*RepositoryGroupPermission, error)
	UpdateRepositoryGroupPermission(ctx context.Context, projectKey, repositorySlug string, permission *RepositoryGroupPermission) error
	ListRepositoryUsersPermission(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*RepositoryUsers, error)
}

// RepositoriesService is a client for communicating with stash repositories endpoints
// Stash API docs: https://docs.atlassian.com/DAC/rest/stash/3.11.3/stash-rest.html
type RepositoriesService service

// Repository represents a stash repository
type Repository struct {
	// Session is the session information for the request.
	Session `json:"sessionInfo,omitempty"`
	// Description is the repository description.
	Description string `json:"description,omitempty"`
	// Forkable is true if the repository is forkable.
	Forkable bool `json:"forkable,omitempty"`
	// HierarchyID is the unique ID of the repository's parent.
	HierarchyID string `json:"hierarchyId,omitempty"`
	// ID is the unique ID of the repository.
	ID float64 `json:"id,omitempty"`
	// Links is the links to other resources.
	Links `json:"links,omitempty"`
	// Name is the repository name.
	Name string `json:"name,omitempty"`
	// Project is the project the repository belongs to.
	Project Project `json:"project,omitempty"`
	// Public is true if the repository is public.
	Public bool `json:"public,omitempty"`
	// ScmID is the unique ID of the repository's SCM.
	ScmID string `json:"scmId,omitempty"`
	// Slug is the unique slug of the repository.
	Slug string `json:"slug,omitempty"`
	// State is the state of the repository.
	State string `json:"state,omitempty"`
	// StatusMessage is the status message of the repository.
	StatusMessage string `json:"statusMessage,omitempty"`
}

// RepositoryList is a list of repositories
type RepositoryList struct {
	// Paging is the paging information for the list of repositories.
	Paging
	// Repositories is the list of repositories.
	Repositories []*Repository `json:"values,omitempty"`
}

// GetRepositories returns the list of repositories permissions for the repository.
func (p *RepositoryList) GetRepositories() []*Repository {
	return p.Repositories
}

// List lists all repositories in a project
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a RepositoryList struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos".
// Accessing personal repositories via REST is achieved through the normal project-centric REST URLs using
// the user's slug prefixed by tilde as the project key.
// example: http://example.com/rest/api/1.0/projects/~johnsmith/repos
func (s *RepositoriesService) List(ctx context.Context, projectKey string, opts *PagingOptions) (*RepositoryList, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list respositories request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list respositories failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	repos := &RepositoryList{
		Repositories: []*Repository{},
	}

	if err := json.Unmarshal(res, repos); err != nil {
		return nil, fmt.Errorf("list repositories failed, unable to unmarshal repository list json: %w", err)
	}

	for _, r := range repos.GetRepositories() {
		r.Session.set(resp)
	}

	return repos, nil
}

// All retrieves all repositories for a given project.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *RepositoriesService) All(ctx context.Context, projectKey string, maxPages int) ([]*Repository, error) {
	if maxPages < 1 {
		maxPages = defaultMaxPages
	}

	r := []*Repository{}
	opts := &PagingOptions{Limit: perPageLimit}
	err := allPages(opts, maxPages, func() (*Paging, error) {
		list, err := s.List(ctx, projectKey, opts)
		if err != nil {
			return nil, err
		}
		r = append(r, list.GetRepositories()...)
		return &list.Paging, nil
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Get returns the repository with the given slug
// Accessing personal repositories via REST is achieved through the normal project-centric REST URLs using
// the user's slug prefixed by tilde as the project key.
// example: http://example.com/rest/api/1.0/projects/~johnsmith/repos/{repositorySlug}
func (s *RepositoriesService) Get(ctx context.Context, projectKey, repoSlug string) (*Repository, error) {
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repoSlug))
	if err != nil {
		return nil, fmt.Errorf("get respository request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get respository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	repo := &Repository{}
	if err := json.Unmarshal(res, repo); err != nil {
		return nil, fmt.Errorf("get repository failed, unable to unmarshall repository json: %w", err)
	}

	repo.Session.set(resp)
	return repo, nil
}

func marshallBody(b interface{}) (io.ReadCloser, error) {
	var body io.ReadCloser
	jsonBody, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	body = io.NopCloser(bytes.NewReader(jsonBody))
	return body, nil
}

// Create creates a new repository
// Create uses the endpoint "POST /rest/api/1.0/projects/{projectKey}/repos".
// The authenticated user must have PROJECT_ADMIN permission for the context project to call this resource.
func (s *RepositoriesService) Create(ctx context.Context, projectKey string, repository *Repository) (*Repository, error) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	body, err := marshallBody(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall repository: %v", err)
	}
	req, err := s.Client.NewRequest(ctx, http.MethodPost, newURI(projectsURI, projectKey, RepositoriesURI), WithBody(body), WithHeader(header))
	if err != nil {
		return nil, fmt.Errorf("create respository request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		if resp.StatusCode == http.StatusConflict {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("create respository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("create repository failed: %s", resp.Status)
	}

	repo := &Repository{}
	if err := json.Unmarshal(res, repo); err != nil {
		return nil, fmt.Errorf("create repository failed, unable to unmarshall repository json: %w", err)
	}

	repo.Session.set(resp)

	return repo, nil
}

// Update updates the repository with the given slug
// The repository's slug is derived from its name. If the name changes the slug may also change.
// Update uses the endpoint "PUT /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}".
func (s *RepositoriesService) Update(ctx context.Context, projectKey, repositorySlug string, repository *Repository) (*Repository, error) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	body, err := marshallBody(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall repository: %v", err)
	}
	req, err := s.Client.NewRequest(ctx, http.MethodPut, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug), WithBody(body), WithHeader(header))
	if err != nil {
		return nil, fmt.Errorf("update repository  request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update repository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("create deploy key for repository failed: %s", resp.Status)
	}

	repo := &Repository{}
	if err := json.Unmarshal(res, repo); err != nil {
		return nil, fmt.Errorf("update repsository failed, unable to unmarshall repository json: %w", err)
	}

	repo.Session.set(resp)

	return repo, nil
}

// Delete deletes the repository with the given slug
// Delete uses the endpoint "DELETE /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}".
func (s *RepositoriesService) Delete(ctx context.Context, projectKey, repoSlug string) error {
	req, err := s.Client.NewRequest(ctx, http.MethodDelete, newURI(projectsURI, projectKey, RepositoriesURI, repoSlug))
	if err != nil {
		return fmt.Errorf("delete repositoryrequest creation failed: %w", err)
	}
	_, _, err = s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("delete repository failed: %w", err)
	}

	return nil
}

// RepositoryGroupPermission is a permission for a given group.
// Repository permissions allow you to manage access to a repository
// beyond that already granted from project permissions.
// The permission is tied to a repository.
// The permission can be either read, write, or admin.
type RepositoryGroupPermission struct {
	// Session is the session information for the request.
	Session `json:"sessionInfo,omitempty"`
	// Group is the group to which the permission applies.
	Group struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	// Permission denotes a group's permission level. Available repository permissions are:
	// REPO_READ
	// REPO_WRITE
	// REPO_ADMIN
	Permission string `json:"permission,omitempty"`
}

// RepositoryGroups represents a list of groups for a given repository.
type RepositoryGroups struct {
	// Paging is the paging information.
	Paging
	// ProjectKey is the project key for the project.
	ProjectKey string `json:"-"`
	// RepositorySlug is the repository slug for the repository.
	RepositorySlug string `json:"-"`
	// Groups is the list of groups permissions.
	Groups []*RepositoryGroupPermission `json:"values,omitempty"`
}

// GetGroups returns the list of groups permissions for the repository.
func (p *RepositoryGroups) GetGroups() []*RepositoryGroupPermission {
	return p.Groups
}

// GetRepositoryGroupPermission retrieve a group that have been granted at least one permission for the specified repository.
// GetRepositoryGroupPermission uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/permissions/groups?filter".
// The authenticated user must have REPO_ADMIN permission for the specified repository to call this resource.
func (s *RepositoriesService) GetRepositoryGroupPermission(ctx context.Context, projectKey, repositorySlug, groupName string) (*RepositoryGroupPermission, error) {
	query := url.Values{
		filterKey: []string{groupName},
	}
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, groupPermisionsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("get group permissions request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get group permissions to repository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	permissions := &RepositoryGroups{}
	if err := json.Unmarshal(res, permissions); err != nil {
		return nil, fmt.Errorf("get group permissions for repository failed, unable to unmarshall repository json: %w", err)
	}

	if len(permissions.Groups) == 0 {
		return nil, ErrNotFound
	}

	permissions.Groups[0].Session.set(resp)
	return permissions.Groups[0], nil
}

// ListRepositoryGroupsPermission retrieve a page of groups that have been granted at least one permission for the specified repository.
// ListRepositoryGroupsPermission uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/permissions/groups?filter".
// The authenticated user must have REPO_ADMIN permission for the specified repository to call this resource.
func (s *RepositoriesService) ListRepositoryGroupsPermission(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*RepositoryGroups, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, groupPermisionsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list group permissions request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list group permissions to repository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	perms := &RepositoryGroups{}
	if err := json.Unmarshal(res, perms); err != nil {
		return nil, fmt.Errorf("list groups permissions for repository failed, unable to unmarshall repository json: %w", err)
	}

	for _, groupPerm := range perms.GetGroups() {
		groupPerm.Session.set(resp)
	}

	return perms, nil
}

// AllGroupsPermission retrieves all repository groups permission.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *RepositoriesService) AllGroupsPermission(ctx context.Context, projectKey, repositorySlug string, maxPages int) ([]*RepositoryGroupPermission, error) {
	if maxPages < 1 {
		maxPages = defaultMaxPages
	}

	p := []*RepositoryGroupPermission{}
	opts := &PagingOptions{Limit: perPageLimit}
	err := allPages(opts, maxPages, func() (*Paging, error) {
		list, err := s.ListRepositoryGroupsPermission(ctx, projectKey, repositorySlug, opts)
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

// UpdateRepositoryGroupPermission Promote or demote a group's permission level for the specified repository.
// UpdateRepositoryGroupPermission uses the endpoint "PUT /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/permissions/groups?permission&name".
func (s *RepositoriesService) UpdateRepositoryGroupPermission(ctx context.Context, projectKey, repositorySlug string, permission *RepositoryGroupPermission) error {
	query := url.Values{
		"name":       []string{permission.Group.Name},
		"permission": []string{permission.Permission},
	}
	req, err := s.Client.NewRequest(ctx, http.MethodPut, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, groupPermisionsURI), WithQuery(query))
	if err != nil {
		return fmt.Errorf("add group permissions request creation failed: %w", err)
	}
	_, resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("add group permissions to repository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return fmt.Errorf("add group permissions to repository failed: %s", resp.Status)
	}

	return nil
}

// RepositoryUserPermission is a permission for a given user.
// Repository permissions allow you to manage access to a repository
// beyond that already granted from project permissions.
// The permission is tied to a repository.
// The permission can be either read, write, or admin.
type RepositoryUserPermission struct {
	// Session is the session information for the request.
	Session    `json:"sessionInfo,omitempty"`
	User       `json:"user,omitempty"`
	Permission string `json:"permission,omitempty"`
}

// RepositoryUsers is a list of users that have been granted at least one permission
// for the specified repository.
type RepositoryUsers struct {
	Paging
	ProjectKey     string                      `json:"-"`
	RepositorySlug string                      `json:"-"`
	Users          []*RepositoryUserPermission `json:"values,omitempty"`
}

// GetUsers return a list of users permissions for the repository.
func (p *RepositoryUsers) GetUsers() []*RepositoryUserPermission {
	return p.Users
}

// ListRepositoryUsersPermission retrieve a page of groups that have been granted at least one permission for the specified repository.
// ListRepositoryUsersPermission uses the endpoint "PUT /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/permissions/users?filter".
// The authenticated user must have REPO_ADMIN permission for the specified repository to call this resource.
func (s *RepositoriesService) ListRepositoryUsersPermission(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*RepositoryUsers, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, userPermisionsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list users permissions request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list users permissions to repository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	users := &RepositoryUsers{}
	if err := json.Unmarshal(res, users); err != nil {
		return nil, fmt.Errorf("list users permissions for repository failed, unable to unmarshall json: %w", err)
	}

	for _, userPerm := range users.GetUsers() {
		userPerm.Session.set(resp)
	}

	return users, nil
}
