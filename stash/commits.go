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
	commitsURI = "commits"
)

// Commits interface defines the methods that can be used to
// retrieve commits of a repository.
type Commits interface {
	List(ctx context.Context, projectKey, repositorySlug, branch string, opts *PagingOptions) (*CommitList, error)
	ListPage(ctx context.Context, projectKey, repositorySlug, branch string, perPage, page int) ([]*CommitObject, error)
	Get(ctx context.Context, projectKey, repositorySlug, commitID string) (*CommitObject, error)
}

// CommitsService is a client for communicating with stash commits endpoint
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
type CommitsService service

// CommitObject represents a commit in stash
type CommitObject struct {
	// Session is the session object for the branch.
	Session `json:"sessionInfo,omitempty"`
	// Author is the author of the commit.
	Author User `json:"author,omitempty"`
	// AuthorTimestamp is the timestamp of the author of the commit.
	AuthorTimestamp int64 `json:"authorTimestamp,omitempty"`
	// Committer is the committer of the commit.
	Committer User `json:"committer,omitempty"`
	// CommitterTimestamp is the timestamp of the committer of the commit.
	CommitterTimestamp int64 `json:"committerTimestamp,omitempty"`
	// DisplayID is the display ID of the commit.
	DisplayID string `json:"displayId,omitempty"`
	// ID is the ID of the commit i.e the SHA1.
	ID string `json:"id,omitempty"`
	// Message is the message of the commit.
	Message string `json:"message,omitempty"`
	// Parents is the list of parents of the commit.
	Parents []*Parent `json:"parents,omitempty"`
}

// Parent represents a parent of a commit.
type Parent struct {
	// DisplayID is the display ID of the commit.
	DisplayID string `json:"displayId,omitempty"`
	// ID is the ID of the commit i.e the SHA1.
	ID string `json:"id,omitempty"`
}

// CommitList represents a list of commits in stash
type CommitList struct {
	// Paging is the paging information.
	Paging
	// Commits is the list of commits.
	Commits []*CommitObject `json:"values,omitempty"`
}

// GetCommits returns the list of commits
func (c *CommitList) GetCommits() []*CommitObject {
	return c.Commits
}

// List returns the list of commits.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a CommitList struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/commits".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *CommitsService) List(ctx context.Context, projectKey, repositorySlug, branch string, opts *PagingOptions) (*CommitList, error) {
	values := url.Values{}
	if branch != "" {
		values.Add("until", branch)
	}
	query := addPaging(values, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, commitsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list commits request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list commits failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("list commits failed: %s", resp.Status)
	}

	c := &CommitList{}
	if err := json.Unmarshal(res, c); err != nil {
		return nil, fmt.Errorf("list commits for repository failed, unable to unmarshall repository json: %w", err)
	}

	for _, commit := range c.GetCommits() {
		commit.Session.set(resp)
	}
	return c, nil
}

// ListPage retrieves all commits for a given page.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *CommitsService) ListPage(ctx context.Context, projectKey, repositorySlug, branch string, perPage, page int) ([]*CommitObject, error) {
	maxPages := 1
	start := 0
	if page > 0 {
		start = (perPage * page) + 1
	}

	p := []*CommitObject{}
	opts := &PagingOptions{Limit: int64(perPage), Start: int64(start)}
	err := allPages(opts, maxPages, func() (*Paging, error) {
		list, err := s.List(ctx, projectKey, repositorySlug, branch, opts)
		if err != nil {
			return nil, err
		}
		p = append(p, list.GetCommits()...)
		return &list.Paging, nil
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Get retrieves a stash commit given it's ID i.e a SHA1.
// Get uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/commits/{commitID}".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *CommitsService) Get(ctx context.Context, projectKey, repositorySlug, commitID string) (*CommitObject, error) {
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, commitsURI, commitID))
	if err != nil {
		return nil, fmt.Errorf("get commit request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get commit failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("get commits failed: %s", resp.Status)
	}

	c := &CommitObject{}
	if err := json.Unmarshal(res, c); err != nil {
		return nil, fmt.Errorf("get commit failed, unable to unmarshall json: %w", err)
	}

	c.Session.set(resp)

	return c, nil
}
