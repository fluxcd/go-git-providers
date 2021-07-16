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
	"net/url"
	"reflect"
	"strconv"

	"github.com/pkg/errors"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
)

var (
	ErrorGetPullRequestMultipleItems = errors.New("multiple items returned for pr name")
)

const (
	PullRequestsURI = "commits"
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

type PullRequest struct {
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

func NewStashPullRequests(client httpclient.ReqResp) StashPullRequests {
	p := &PullRequests{
		Paging:       Paging{},
		ReqResp:      client,
		PullRequests: make([]*PullRequest, 0),
	}
	return p
}

type StashPullRequests interface {
	List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName, repoName, prID string) (*PullRequest, error)
	Update(ctx context.Context, projectName, repoName string, pr *PullRequest) (*PullRequest, error)
	Create(ctx context.Context, projectName, repoName string, pr *PullRequest) (*PullRequest, error)
	GetPullRequests() []*PullRequest
}

type PullRequests struct {
	StashPullRequests
	Paging
	httpclient.ReqResp
	PullRequests []*PullRequest `json:"values,omitempty"`
}

func (p *PullRequests) GetPullRequests() []*PullRequest {
	return p.PullRequests
}

func (p *PullRequests) List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI), query, nil, nil, nil)
	if err != nil {
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list PullRequests\n")

	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal project list\n")
	}

	return &p.Paging, nil
}

func (p *PullRequests) Get(ctx context.Context, projectName, repoName, prID string) (*PullRequest, error) {
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI, prID), nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to get commitsitory\n")
	}
	commit := &PullRequest{}
	if err := json.Unmarshal([]byte(resp), &commit); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshall commitsitory json\n")
	}
	return commit, nil
}

func (p *PullRequests) Create(ctx context.Context, projectName, repoName string, pr *PullRequest) (*PullRequest, error) {
	hdr := httpclient.Header{"Content-Type": "application/json"}

	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI), nil, pr, &httpclient.Post, &hdr)
	if err != nil {
		if code == 409 {
			return nil, gitprovider.ErrAlreadyExists
		}
		return nil, err
	}
	if code != 201 {
		return nil, errors.WithMessagef(err, "failed to create repository\n")
	}

	if err := json.Unmarshal([]byte(resp), pr); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshall repository json\n")
	}
	return pr, nil
}

func (p *PullRequests) Update(ctx context.Context, projectName, repoName string, pr *PullRequest) (*PullRequest, error) {
	hdr := httpclient.Header{"Content-Type": "application/json"}

	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, PullRequestsURI, strconv.Itoa(pr.ID)), nil, pr, &httpclient.Put, &hdr)
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, errors.WithMessagef(err, "failed to update repository\n")
	}

	if err := json.Unmarshal([]byte(resp), pr); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshall repository json\n")
	}
	return pr, nil
}

type PullRequestGroupPermission struct {
	Group struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	Permission string `json:"permission,omitempty"`
}
