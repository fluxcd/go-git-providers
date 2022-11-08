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
	"strconv"
)

const (
	pullRequestsURI = "pull-requests"
	mergeURI        = "merge"
)

// PullRequests interface defines the methods that can be used to
// retrieve pull requests of a repository.
type PullRequests interface {
	Get(ctx context.Context, projectKey, repositorySlug string, prID int) (*PullRequest, error)
	List(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*PullRequestList, error)
	All(ctx context.Context, projectKey, repositorySlug string) ([]*PullRequest, error)
	Create(ctx context.Context, projectKey, repositorySlug string, pr *CreatePullRequest) (*PullRequest, error)
	Update(ctx context.Context, projectKey, repositorySlug string, pr *PullRequest) (*PullRequest, error)
	Merge(ctx context.Context, projectKey, repositorySlug string, prID int, version int) (*PullRequest, error)
	Delete(ctx context.Context, projectKey, repositorySlug string, IDVersion IDVersion) error
}

// PullRequestsService is a client for communicating with stash pull requests endpoint
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
type PullRequestsService service

// Participant is a participant of a pull request
type Participant struct {
	// Approved indicates if the participant has approved the pull request
	Approved bool `json:"approved,omitempty"`
	// Role indicates the role of the participant
	Role string `json:"role,omitempty"`
	// Status indicates the status of the participant
	Status string `json:"status,omitempty"`
	// User is the participant
	User `json:"user,omitempty"`
}

// Ref represents a git reference
type Ref struct {
	// DisplayID is the reference name
	DisplayID string `json:"displayId,omitempty"`
	// ID is the reference id i.e a git reference
	ID string `json:"id,omitempty"`
	// LatestCommit is the latest commit of the reference
	LatestCommit string `json:"latestCommit,omitempty"`
	// Repository is the repository of the reference
	Repository `json:"repository,omitempty"`
	// Type is the type of the reference
	Type string `json:"type,omitempty"`
}

// CreatePullRequest creates a pull request from
// a source branch or tag to a target branch.
type CreatePullRequest struct {
	// Closed indicates if the pull request is closed
	Closed bool `json:"closed,omitempty"`
	// Description is the description of the pull request
	Description string `json:"description,omitempty"`
	// FromRef is the source branch or tag
	FromRef Ref `json:"fromRef,omitempty"`
	// Locked indicates if the pull request is locked
	Locked bool `json:"locked,omitempty"`
	// Open indicates if the pull request is open
	Open bool `json:"open,omitempty"`
	// State is the state of the pull request
	State string `json:"state,omitempty"`
	// Title is the title of the pull request
	Title string `json:"title,omitempty"`
	// ToRef is the target branch
	ToRef Ref `json:"toRef,omitempty"`
	// Reviewers is the list of reviewers
	Reviewers []User `json:"reviewers,omitempty"`
}

// IDVersion is a pull request id and version
type IDVersion struct {
	// ID is the id of the pull request
	ID int `json:"id"`
	// Version is the version of the pull request
	Version int `json:"version"`
}

// PullRequest is a pull request
type PullRequest struct {
	// Session is the session of the pull request
	Session `json:"sessionInfo,omitempty"`
	// Author is the author of the pull request
	Author *Participant `json:"author,omitempty"`
	// Closed indicates if the pull request is closed
	Closed bool `json:"closed,omitempty"`
	// CreatedDate is the creation date of the pull request
	CreatedDate int64 `json:"createdDate,omitempty"`
	// Description is the description of the pull request
	Description string `json:"description,omitempty"`
	// FromRef is the source branch or tag
	FromRef Ref `json:"fromRef,omitempty"`
	IDVersion
	// Links is a set of hyperlinks that link to other related resources.
	Links `json:"links,omitempty"`
	// Locked indicates if the pull request is locked
	Locked bool `json:"locked,omitempty"`
	// Open indicates if the pull request is open
	Open bool `json:"open,omitempty"`
	// Participants are the participants of the pull request
	Participants []Participant `json:"participants,omitempty"`
	// Properties are the properties of the pull request
	Properties Properties `json:"properties,omitempty"`
	// Reviewers are the reviewers of the pull request
	Reviewers []Participant `json:"reviewers,omitempty"`
	// State is the state of the pull request
	State string `json:"state,omitempty"`
	// Title is the title of the pull request
	Title string `json:"title,omitempty"`
	// ToRef is the target branch
	ToRef Ref `json:"toRef,omitempty"`
	// UpdatedDate is the update date of the pull request
	UpdatedDate int64 `json:"updatedDate,omitempty"`
}

// Properties are the properties of a pull request
type Properties struct {
	// MergeResult is the merge result of the pull request
	MergeResult MergeResult `json:"mergeResult,omitempty"`
	// OpenTaskCount is the number of open tasks
	OpenTaskCount float64 `json:"openTaskCount,omitempty"`
	// ResolvedTaskCount is the number of resolved tasks
	ResolvedTaskCount float64 `json:"resolvedTaskCount,omitempty"`
}

// MergeResult is the merge result of a pull request
type MergeResult struct {
	// Current is the current merge result
	Current bool `json:"current,omitempty"`
	// Outcome is the outcome of the merge
	Outcome string `json:"outcome,omitempty"`
}

// PullRequestList is a list of pull requests
type PullRequestList struct {
	// Paging is the paging information
	Paging
	// PullRequests are the pull requests
	PullRequests []*PullRequest `json:"values,omitempty"`
}

// GetPullRequests returns a list of pull requests
func (p *PullRequestList) GetPullRequests() []*PullRequest {
	return p.PullRequests
}

// List returns the list of pull requests.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a PullRequestsList struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *PullRequestsService) List(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*PullRequestList, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, pullRequestsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list pull requests request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list pull requests failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	p := &PullRequestList{}
	if err := json.Unmarshal(res, p); err != nil {
		return nil, fmt.Errorf("list pull requests failed, unable to unmarshal pull request list json: %w", err)
	}

	for _, r := range p.GetPullRequests() {
		r.Session.set(resp)
	}

	return p, nil
}

// All retrieves all pull requests for a given repository.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *PullRequestsService) All(ctx context.Context, projectKey, repositorySlug string) ([]*PullRequest, error) {
	pr := []*PullRequest{}
	opts := &PagingOptions{Limit: perPageLimit}
	err := allPages(opts, func() (*Paging, error) {
		list, err := s.List(ctx, projectKey, repositorySlug, opts)
		if err != nil {
			return nil, err
		}
		pr = append(pr, list.GetPullRequests()...)
		return &list.Paging, nil
	})
	if err != nil {
		return nil, err
	}

	return pr, nil
}

// Get retrieves a pull request given it's ID.
// Get uses the endpoint "GET /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests/{pullRequestId}".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *PullRequestsService) Get(ctx context.Context, projectKey, repositorySlug string, prID int) (*PullRequest, error) {
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, pullRequestsURI, strconv.Itoa(prID)))
	if err != nil {
		return nil, fmt.Errorf("get pull request request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get pull request failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	p := &PullRequest{}
	if err := json.Unmarshal(res, p); err != nil {
		return nil, fmt.Errorf("get pull request failed, unable to unmarshal pull request json: %w", err)
	}

	p.Session.set(resp)

	return p, nil
}

// Create creates a pull request.
// Create uses the endpoint "POST /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests".
func (s *PullRequestsService) Create(ctx context.Context, projectKey, repositorySlug string, pr *CreatePullRequest) (*PullRequest, error) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	body, err := marshallBody(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall pull request: %v", err)
	}
	req, err := s.Client.NewRequest(ctx, http.MethodPost, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, pullRequestsURI), WithBody(body), WithHeader(header))
	if err != nil {
		return nil, fmt.Errorf("create pull request request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create pull request failed: %w", err)
	}

	p := &PullRequest{}
	if err := json.Unmarshal(res, p); err != nil {
		return nil, fmt.Errorf("create pull request failed, unable to unmarshal pull json: %w", err)
	}

	p.Session.set(resp)

	return p, nil
}

// Update updates the pull request with the given ID
// Update uses the endpoint "PUT /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests/{pullRequestId}".
func (s *PullRequestsService) Update(ctx context.Context, projectKey, repositorySlug string, pr *PullRequest) (*PullRequest, error) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	body, err := marshallBody(pr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall pull request: %v", err)
	}

	req, err := s.Client.NewRequest(ctx, http.MethodPut, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, pullRequestsURI, strconv.Itoa(pr.ID)), WithBody(body), WithHeader(header))
	if err != nil {
		return nil, fmt.Errorf("update pull request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update pull failed: %w", err)
	}

	if resp != nil && resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update failed with status code %d, error: %s", resp.StatusCode, res)
	}

	p := &PullRequest{}
	if err := json.Unmarshal(res, p); err != nil {
		return nil, fmt.Errorf("create pull request failed, unable to unmarshal pull request json: %w", err)
	}

	p.Session.set(resp)

	return p, nil
}

// Merge the pull request with the given ID and version.
// Merge uses the endpoint "POST /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests/{pullRequestId}/merge?version".
func (s *PullRequestsService) Merge(ctx context.Context, projectKey, repositorySlug string, prID int, version int) (*PullRequest, error) {
	query := url.Values{
		"version": []string{strconv.Itoa(version)},
	}

	header := http.Header{"X-Atlassian-Token": []string{"no-check"}}

	req, err := s.Client.NewRequest(ctx, http.MethodPost, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, pullRequestsURI, strconv.Itoa(prID), mergeURI), WithQuery(query), WithHeader(header))
	if err != nil {
		return nil, fmt.Errorf("merge pull request request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("merge pull request failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("list commits failed: %s", resp.Status)
	}

	p := &PullRequest{}
	if err := json.Unmarshal(res, p); err != nil {
		return nil, fmt.Errorf("merge pull  request failed, unable to unmarshal pull request json: %w", err)
	}

	p.Session.set(resp)

	return p, nil
}

// Delete deletes the pull request with the given ID
// Delete uses the endpoint "DELETE /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/pull-requests/{pullRequestId}".
// To call this resource, users must:
// - be the pull request author, if the system is configured to allow authors to delete their own pull requests (this is the default) OR
// - have repository administrator permission for the repository the pull request is targeting
// A body containing the ID and version of the pull request must be provided with this request.
//
//	{
//	  "id": 1,
//	  "version": 1
//	}
func (s *PullRequestsService) Delete(ctx context.Context, projectKey, repositorySlug string, IDVersion IDVersion) error {
	header := http.Header{"Content-Type": []string{"application/json"}}
	body, err := marshallBody(IDVersion.Version)
	req, err := s.Client.NewRequest(ctx, http.MethodDelete, newURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, pullRequestsURI, strconv.Itoa(IDVersion.ID)), WithBody(body), WithHeader(header))
	if err != nil {
		return fmt.Errorf("delete pull request frequest creation failed: %w", err)
	}
	_, resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("delete pull request for repository failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	return nil
}
