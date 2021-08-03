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
	ErrorGetBranchMultipleItems = errors.New("multiple items returned for pr name")
)

const (
	branchesURI      = "branches"
	defaultBranchURI = "default"
)

type Branch struct {
	SessionInfo     `json:"sessionInfo,omitempty"`
	DisplayId       string `json:"displayId,omitempty"`
	ID              string `json:"id,omitempty"`
	IsDefault       bool   `json:"isDefault,omitempty"`
	LatestChangeset string `json:"latestChangeset,omitempty"`
	LatestBranch    string `json:"latestBranch,omitempty"`
	Type            string `json:"type,omitempty"`
}

func (r *Branch) Default() {
}

func (r *Branch) ValidateInfo() error {
	return nil
}

func (r *Branch) Equals(actual gitprovider.InfoRequest) bool {
	return reflect.DeepEqual(r, actual)
}

func NewStashBranches(client *stashClientImpl) StashBranches {
	p := &Branches{
		Paging:    Paging{},
		Requester: client.Client(),
		Branches:  make([]*Branch, 0),
		log:       client.log,
	}
	return p
}

type StashBranches interface {
	List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName, repoName, branchID string) (*Branch, error)
	Default(ctx context.Context, projectName, repoName string) (*Branch, error)
	getBranches() []*Branch
}

type Branches struct {
	StashBranches
	Paging
	httpclient.Requester
	Branches []*Branch `json:"values,omitempty"`
	log      logr.Logger
}

func (p *Branches) getBranches() []*Branch {
	return p.Branches
}

func (p *Branches) List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, branchesURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list branches for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list branches for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list branches for repository failed, unable to obtain response body, %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list branches for repository failed, unable to unmarshall repository json, %w", err)
	}

	for _, branches := range p.getBranches() {
		branches.setSessionInfo(resp)
	}
	return &p.Paging, nil
}

func (p *Branches) Get(ctx context.Context, projectName, repoName, branchID string) (*Branch, error) {
	var query *url.Values
	query = addPaging(query, &ListOptions{})
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, branchesURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get branch failed, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get branch for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get branch failed, unable to obtain response body, %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("get branch for repository failed, unable to unmarshall repository json, %w", err)
	}

	for _, branch := range p.getBranches() {
		if branch.DisplayId == branchID {
			branch.setSessionInfo(resp)
			return branch, nil
		}
	}

	return nil, gitprovider.ErrNotFound
}

func (p *Branches) Default(ctx context.Context, projectName, repoName string) (*Branch, error) {
	var query *url.Values
	query = addPaging(query, &ListOptions{})
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, branchesURI, defaultBranchURI), nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get branch failed, %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get branches for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get branch failed, unable to obtain response body, %w", err)
	}

	defaultBranch := &Branch{}
	if err := json.Unmarshal([]byte(respBody), defaultBranch); err != nil {
		return nil, fmt.Errorf("list branches for repository failed, unable to unmarshall repository json, %w", err)
	}
	defaultBranch.setSessionInfo(resp)

	return defaultBranch, gitprovider.ErrNotFound
}
