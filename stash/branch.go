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
	branchesURI      = "branches"
	defaultBranchURI = "default"
)

// Branches interface defines the methods that can be used to
// retrieve branches of a repository.
type Branches interface {
	List(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*BranchList, error)
	Get(ctx context.Context, projectKey, repositorySlug, branchID string) (*Branch, error)
	Create(ctx context.Context, projectKey, repositorySlug, branchID, startPoint string) (*Branch, error)
	Default(ctx context.Context, projectKey, repositorySlug string) (*Branch, error)
	SetDefault(ctx context.Context, projectKey, repositorySlug, branchID string) error
}

// BranchesService is a client for communicating with stash branches endpoint
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
type BranchesService service

// Branch represents a branch of a repository.
type Branch struct {
	// Session is the session object for the branch.
	Session `json:"sessionInfo,omitempty"`
	// DisplayID is the branch name e.g. main.
	DisplayID string `json:"displayId,omitempty"`
	// ID is the branch reference e.g. refs/heads/main.
	ID string `json:"id,omitempty"`
	// IsDefault is true if this is the default branch.
	IsDefault bool `json:"isDefault,omitempty"`
	// LatestChangeset is the latest changeset on this branch.
	LatestChangeset string `json:"latestChangeset,omitempty"`
	// LatestCommit is the latest commit on this branch.
	LatestCommit string `json:"latestCommit,omitempty"`
	// Type is the type of branch.
	Type string `json:"type,omitempty"`
}

// BranchList is a list of branches.
type BranchList struct {
	// Paging is the paging information.
	Paging
	// Branches is the list of branches.
	Branches []*Branch `json:"values,omitempty"`
}

// GetBranches returns the list of branches.
func (b *BranchList) GetBranches() []*Branch {
	return b.Branches
}

// List returns the list of branches.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a BranchList struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/branches".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *BranchesService) List(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*BranchList, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, branchesURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list branches request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list branches failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	b := &BranchList{}

	if err := json.Unmarshal(res, b); err != nil {
		return nil, fmt.Errorf("list branches for repository failed, unable to unmarshall repository json: %w", err)
	}

	for _, branches := range b.GetBranches() {
		branches.Session.set(resp)
	}

	return b, nil
}

// Get retrieves a stash branch given it's ID i.e a git reference.
// Get uses the endpoint
// "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/branches?base&details&filterText&orderBy".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *BranchesService) Get(ctx context.Context, projectKey, repositorySlug, branchID string) (*Branch, error) {
	query := url.Values{
		"filterText": []string{branchID},
	}

	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, branchesURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("get branch request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get branch failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	b := &Branch{}
	if err := json.Unmarshal(res, b); err != nil {
		return nil, fmt.Errorf("get branch for repository failed, unable to unmarshall repository json: %w", err)
	}

	b.Session.set(resp)
	return b, nil

}

// Default retrieves the default branch of a repository.
// Default uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/branches/default".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *BranchesService) Default(ctx context.Context, projectKey, repositorySlug string) (*Branch, error) {
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, branchesURI, defaultBranchURI))
	if err != nil {
		return nil, fmt.Errorf("get branch request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get branch failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	b := &Branch{}
	if err := json.Unmarshal(res, b); err != nil {
		return nil, fmt.Errorf("list branches for repository failed, unable to unmarshall repository json: %w", err)
	}

	b.Session.set(resp)

	return b, nil
}

// SetDefault updates the default branch of a repository.
// SetDefault uses the endpoint "PUT /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/branches/default".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *BranchesService) SetDefault(ctx context.Context, projectKey, repositorySlug, branchID string) error {
	id := struct {
		ID string `json:"id"`
	}{
		ID: branchID,
	}
	body, err := marshallBody(id)
	header := http.Header{"Content-Type": []string{"application/json"}}

	if err != nil {
		return fmt.Errorf("failed to marshall branch id: %v", err)
	}
	req, err := s.Client.NewRequest(ctx, http.MethodPut, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, branchesURI, defaultBranchURI), WithBody(body), WithHeader(header))
	if err != nil {
		return fmt.Errorf("set default branch request creation failed: %w", err)
	}
	_, resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("set default branch failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	return nil
}

// Create creates a branch for a repository.
// It uses the branchID as the name of the branch and startPoint as the commit to start from.
// Create uses the endpoint "POST /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/branches".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *BranchesService) Create(ctx context.Context, projectKey, repositorySlug, branchID, startPoint string) (*Branch, error) {
	branch := struct {
		Name       string `json:"name"`
		StartPoint string `json:"startPoint"`
	}{
		Name:       branchID,
		StartPoint: startPoint,
	}
	body, err := marshallBody(branch)
	header := http.Header{"Content-Type": []string{"application/json"}}

	if err != nil {
		return nil, fmt.Errorf("failed to marshall branch: %v", err)
	}
	req, err := s.Client.NewRequest(ctx, http.MethodPost, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, branchesURI), WithBody(body), WithHeader(header))
	if err != nil {
		return nil, fmt.Errorf("create branch request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create branch failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	b := &Branch{}
	if err := json.Unmarshal(res, b); err != nil {
		return nil, fmt.Errorf("create branch for repository failed, unable to unmarshall branch json: %w", err)
	}

	b.Session.set(resp)
	return b, nil
}
