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
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/pkg/errors"

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

func NewStashCommits(client httpclient.ReqResp) StashCommits {
	p := &Commits{
		Paging:  Paging{},
		ReqResp: client,
		Commits: make([]*Commit, 0),
	}
	return p
}

type StashCommits interface {
	List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName, repoName, commitID string) (*Commit, error)
	GetCommits() []*Commit
}

type Commits struct {
	StashCommits
	Paging
	httpclient.ReqResp
	Commits []*Commit `json:"values,omitempty"`
}

func (p *Commits) GetCommits() []*Commit {
	return p.Commits
}

func (p *Commits) List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, CommitsURI), query, nil, nil, nil)
	if err != nil {
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.WithMessagef(err, "failed to list Commits\n")

	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to retrieve response body, error: %w\n", err))
	}

	if debug {
		fmt.Printf("Response, Status: %d, Body...%s\n", resp.StatusCode, respBody)
	}
	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal project list\n")
	}

	return &p.Paging, nil
}

func (p *Commits) Get(ctx context.Context, projectName, repoName, commitID string) (*Commit, error) {
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName, CommitsURI, commitID), nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to get commitsitory\n")
	}
	commit := &Commit{}
	if err := json.Unmarshal([]byte(resp), &commit); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshall commitsitory json\n")
	}
	return commit, nil
}

type CommitGroupPermission struct {
	Group struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func NewStashCommitGroups(client httpclient.ReqResp, projectName, commitsitoryName string) StashCommitGroups {
	p := &CommitGroups{
		Paging:      Paging{},
		ReqResp:     client,
		ProjectName: projectName,
		CommitName:  commitsitoryName,
		Groups:      make([]*CommitGroupPermission, 0),
	}
	return p
}

type StashCommitGroups interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, groupName string) (*CommitGroupPermission, error)
	Create(ctx context.Context, permission *CommitGroupPermission) error
	getGroups() []*CommitGroupPermission
}

type CommitGroups struct {
	StashCommits
	Paging
	httpclient.ReqResp
	ProjectName string                   `json:"-"`
	CommitName  string                   `json:"-"`
	Groups      []*CommitGroupPermission `json:"values,omitempty"`
}

func (p *CommitGroups) getGroups() []*CommitGroupPermission {
	return p.Groups
}

func (p *CommitGroups) Create(ctx context.Context, permission *CommitGroupPermission) error {
	var query *url.Values = &url.Values{
		"name":       []string{permission.Group.Name},
		"permission": []string{permission.Permission},
	}
	code, _, err := p.ReqResp.Do(ctx, newURI(projectsURI, p.ProjectName, CommitsURI, p.CommitName, groupPermisionsURI), query, nil, &httpclient.Put, nil)
	if err != nil {
		return err
	}
	if code != 204 {
		return errors.WithMessagef(err, "failed to set commitsitory permission for group\n")
	}
	return nil
}

func (p *CommitGroups) Get(ctx context.Context, groupName string) (*CommitGroupPermission, error) {
	var query *url.Values = &url.Values{
		filterKey: []string{groupName},
	}
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, p.ProjectName, CommitsURI, p.CommitName, groupPermisionsURI), query, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to get commitsitory permission for group\n")
	}
	permissions := &CommitGroups{}
	if err := json.Unmarshal([]byte(resp), &permissions); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshall commitsitory permissions for group json\n")
	}
	for _, groupPerm := range permissions.Groups {
		if groupPerm.Group.Name == groupName {
			return groupPerm, nil
		}
	}
	return nil, gitprovider.ErrNotFound
}

func (p *CommitGroups) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, p.ProjectName, CommitsURI, p.CommitName, groupPermisionsURI), query, nil, nil, nil)
	if err != nil {
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list commitsitories\n")

	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal commitsitory list\n")
	}

	return &p.Paging, nil
}

type CommitUserPermission struct {
	User       User   `json:"user,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func NewStashCommitUsers(client httpclient.ReqResp, commitsitoryName string) StashCommitUsers {
	p := &CommitUsers{
		Paging:     Paging{},
		ReqResp:    client,
		CommitName: commitsitoryName,
		Users:      make([]*CommitUserPermission, 0),
	}
	return p
}

type StashCommitUsers interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	//GetMembers(ctx context.Context, commitsitoryName string) (*Commit, error)
	getUsers() []*CommitUserPermission
}

type CommitUsers struct {
	StashCommits
	Paging
	httpclient.ReqResp
	ProjectName string                  `json:"-"`
	CommitName  string                  `json:"-"`
	Users       []*CommitUserPermission `json:"values,omitempty"`
}

func (p *CommitUsers) getUsers() []*CommitUserPermission {
	return p.Users
}

func (p CommitUsers) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, p.ProjectName, userPermisionsURI), query, nil, nil, nil)
	if err != nil {
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list commitsitories\n")

	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal commitsitory list\n")
	}

	return &p.Paging, nil
}
