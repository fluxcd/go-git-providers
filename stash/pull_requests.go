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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/go-logr/logr"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
)

var (
	ErrorGetPullRequestMultipleItems = errors.New("multiple items returned for pr name")
)

const (
	PullRequestsURI = "pull-requests"
)

type Participant struct {
	Approved bool   `json:"approved,omitempty"`
	Role     string `json:"role,omitempty"`
	Status   string `json:"status,omitempty"`
	User     `json:"user,omitempty"`
}

type Ref struct {
	DisplayId    string `json:"displayId,omitempty"`
	ID           string `json:"id,omitempty"`
	LatestCommit string `json:"latestCommit,omitempty"`
	Repository   `json:"repository,omitempty"`
	Type         string `json:"type,omitempty"`
}

type PullRequestCreation struct {
	Closed      bool   `json:"closed,omitempty"`
	Description string `json:"description,omitempty"`
	FromRef     Ref    `json:"fromRef,omitempty"`
	Locked      bool   `json:"locked,omitempty"`
	Open        bool   `json:"open,omitempty"`
	State       string `json:"state,omitempty"`
	Title       string `json:"title,omitempty"`
	ToRef       Ref    `json:"toRef,omitempty"`
}

type PullRequest struct {
	SessionInfo  `json:"sessionInfo,omitempty"`
	Author       Participant `json:"author,omitempty"`
	Closed       bool        `json:"closed,omitempty"`
	CreatedDate  int64       `json:"createdDate,omitempty"`
	Description  string      `json:"description,omitempty"`
	FromRef      Ref         `json:"fromRef,omitempty"`
	ID           int         `json:"id,omitempty"`
	Links        `json:"links,omitempty"`
	Locked       bool          `json:"locked,omitempty"`
	Open         bool          `json:"open,omitempty"`
	Participants []Participant `json:"participants,omitempty"`
	Properties   struct {
		MergeResult struct {
			Current bool   `json:"current,omitempty"`
			Outcome string `json:"outcome,omitempty"`
		} `json:"mergeResult,omitempty"`
		OpenTaskCount     float64 `json:"openTaskCount,omitempty"`
		ResolvedTaskCount float64 `json:"resolvedTaskCount,omitempty"`
	} `json:"properties,omitempty"`
	Reviewers   []Participant `json:"reviewers,omitempty"`
	State       string        `json:"state,omitempty"`
	Title       string        `json:"title,omitempty"`
	ToRef       Ref           `json:"toRef,omitempty"`
	UpdatedDate int64         `json:"updatedDate,omitempty"`
	Version     int64         `json:"version,omitempty"`
}

func (r *PullRequest) Default() {
}

func (r *PullRequest) ValidateInfo() error {
	return nil
}

func (r *PullRequest) Equals(actual gitprovider.InfoRequest) bool {
	return reflect.DeepEqual(r, actual)
}

func NewStashPullRequests(client *stashClientImpl) StashPullRequests {
	p := &PullRequests{
		Paging:       Paging{},
		Requester:    client.Client(),
		PullRequests: make([]*PullRequest, 0),
		log:          client.log,
	}
	return p
}

type StashPullRequests interface {
	List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName, repoName string, prID int) (*PullRequest, error)
	Delete(ctx context.Context, projectName, repoName string, prID int) error
	Update(ctx context.Context, projectName, repoName string, pr *PullRequest) (*PullRequest, error)
	Create(ctx context.Context, projectName, repoName string, pr *PullRequestCreation) (*PullRequest, error)
	getPullRequests() []*PullRequest
}

type PullRequests struct {
	StashPullRequests
	Paging
	httpclient.Requester
	PullRequests []*PullRequest `json:"values,omitempty"`
	log          logr.Logger
}

func (p *PullRequests) getPullRequests() []*PullRequest {
	return p.PullRequests
}

func (p *PullRequests) List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list pull requests failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list pull requests api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list pull requests failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list pull requests failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range p.getPullRequests() {
		r.setSessionInfo(resp)
	}

	return &p.Paging, nil
}

func (p *PullRequests) Get(ctx context.Context, projectName, repoName string, prID int) (*PullRequest, error) {
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI, strconv.Itoa(prID)), nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get pull request failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pull request api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get pull request failed, unable to obtain response body: %w", err)
	}

	commit := &PullRequest{}
	if err := json.Unmarshal([]byte(respBody), &commit); err != nil {
		return nil, fmt.Errorf("get pull request failed, unable to unmarshal repository list json, %w", err)
	}

	commit.setSessionInfo(resp)

	return commit, nil
}

func (p *PullRequests) Delete(ctx context.Context, projectName, repoName string, prID int) error {
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI, strconv.Itoa(prID)), nil, nil, &httpclient.Delete, nil)
	if err != nil {
		return fmt.Errorf("delete pull request for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete pull request for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (p *PullRequests) Create(ctx context.Context, projectName, repoName string, pr *PullRequestCreation) (*PullRequest, error) {
	hdr := http.Header{"Content-Type": []string{"application/json"}}

	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI), nil, pr, &httpclient.Post, &hdr)
	if err != nil {
		if resp.StatusCode == http.StatusConflict {
			return nil, gitprovider.ErrAlreadyExists
		}
		return nil, fmt.Errorf("create pull request failed, %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create pull request api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("create pull request failed, unable to obtain response body: %w", err)
	}

	commit := &PullRequest{}
	if err := json.Unmarshal([]byte(respBody), &commit); err != nil {
		return nil, fmt.Errorf("create pull request failed, unable to unmarshal repository list json, %w", err)
	}

	commit.setSessionInfo(resp)

	return commit, nil
}

func (p *PullRequests) Update(ctx context.Context, projectName, repoName string, pr *PullRequest) (*PullRequest, error) {
	hdr := http.Header{"Content-Type": []string{"application/json"}}

	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI, strconv.Itoa(pr.ID)), nil, pr, &httpclient.Put, &hdr)
	if err != nil {
		return nil, fmt.Errorf("update pull request failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("update pull request api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("create pull request failed, unable to obtain response body: %w", err)
	}

	commit := &PullRequest{}
	if err := json.Unmarshal([]byte(respBody), &commit); err != nil {
		return nil, fmt.Errorf("create pull request failed, unable to unmarshal repository list json, %w", err)
	}

	commit.setSessionInfo(resp)

	return commit, nil
}

type PullRequestGroupPermission struct {
	Group struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	Permission string `json:"permission,omitempty"`
}
