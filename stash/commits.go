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

	"github.com/go-logr/logr"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
)

var (
	ErrorGetCommitMultipleItems = errors.New("multiple items returned for commit name")
)

const (
	CommitsURI = "commits"
)

type Commit struct {
	SessionInfo        `json:"sessionInfo,omitempty"`
	Author             User   `json:"author,omitempty"`
	AuthorTimestamp    int64  `json:"authorTimestamp,omitempty"`
	Committer          User   `json:"committer,omitempty"`
	CommitterTimestamp int64  `json:"committerTimestamp,omitempty"`
	DisplayId          string `json:"displayId,omitempty"`
	ID                 string `json:"id,omitempty"`
	Message            string `json:"message,omitempty"`
	Parents            []struct {
		DisplayId string `json:"displayId,omitempty"`
		ID        string `json:"id,omitempty"`
	} `json:"parents,omitempty"`
}

func (r *Commit) Default() {
}

func (r *Commit) ValidateInfo() error {
	return nil
}

func (r *Commit) Equals(actual gitprovider.InfoRequest) bool {
	return reflect.DeepEqual(r, actual)
}

func NewStashCommits(client *stashClientImpl) StashCommits {
	p := &Commits{
		Paging:  Paging{},
		ReqResp: client.Client(),
		Commits: make([]*Commit, 0),
		log:     client.log,
	}
	return p
}

type StashCommits interface {
	List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName, repoName, commitID string) (*Commit, error)
	getCommits() []*Commit
}

type Commits struct {
	StashCommits
	Paging
	httpclient.ReqResp
	Commits []*Commit `json:"values,omitempty"`
	log     logr.Logger
}

func (p *Commits) getCommits() []*Commit {
	return p.Commits
}

func (p *Commits) List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, CommitsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list commits for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list commits for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list commits for repository failed, unable to obtain response body, %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list commits for repository failed, unable to unmarshall repository json, %w", err)
	}

	for _, commits := range p.getCommits() {
		commits.setSessionInfo(resp)
	}
	return &p.Paging, nil
}

func (p *Commits) Get(ctx context.Context, projectName, repoName, commitID string) (*Commit, error) {
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, CommitsURI, commitID), nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get commit failed, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get commits for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get commit failed, unable to obtain response body, %w", err)
	}

	commit := &Commit{}
	if err := json.Unmarshal([]byte(respBody), commit); err != nil {
		return nil, fmt.Errorf("get commit failed, unable to unmarshall json, %w", err)
	}

	commit.setSessionInfo(resp)

	return commit, nil
}
