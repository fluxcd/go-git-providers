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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
	"github.com/go-logr/logr"
)

var (
	ErrorGetRepositoryMultipleItems = errors.New("multiple items returned for repo name")
)

const (
	RepositoriesURI = "repos"
)

type Repository struct {
	SessionInfo   `json:"sessionInfo,omitempty"`
	Description   string  `json:"description,omitempty"`
	Forkable      bool    `json:"forkable,omitempty"`
	HierarchyId   string  `json:"hierarchyId,omitempty"`
	ID            float64 `json:"id,omitempty"`
	Links         `json:"links,omitempty"`
	Name          string  `json:"name,omitempty"`
	Project       Project `json:"project,omitempty"`
	Public        bool    `json:"public,omitempty"`
	ScmId         string  `json:"scmId,omitempty"`
	Slug          string  `json:"slug,omitempty"`
	State         string  `json:"state,omitempty"`
	StatusMessage string  `json:"statusMessage,omitempty"`
}

func (r *Repository) Default() {
}

func (r *Repository) ValidateInfo() error {
	return nil
}

func (r *Repository) Equals(actual gitprovider.InfoRequest) bool {
	return reflect.DeepEqual(r, actual)
}

func NewStashRepositories(client *stashClientImpl) stashRepositories {
	p := &Repositories{
		Paging:       Paging{},
		Requester:    client.Client(),
		Repositories: make([]*Repository, 0),
		log:          client.log,
	}
	return p
}

type stashRepositories interface {
	List(ctx context.Context, projectName string, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName, repoName string) (*Repository, error)
	getRepositories() []*Repository
	Create(ctx context.Context, projectName string, repo *Repository) (*Repository, error)
	Update(ctx context.Context, projectName string, repo *Repository) (*Repository, error)
	Delete(ctx context.Context, projectName, repoName string) error
}

type Repositories struct {
	stashRepositories
	Paging
	httpclient.Requester
	Repositories []*Repository `json:"values,omitempty"`
	log          logr.Logger
}

func (p *Repositories) getRepositories() []*Repository {
	return p.Repositories
}

func (p *Repositories) List(ctx context.Context, projectName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list respositories failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list repositories api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list repositories failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list repositories failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range p.getRepositories() {
		r.setSessionInfo(resp)
	}

	return &p.Paging, nil
}

func (p *Repositories) Get(ctx context.Context, projectName, repoName string) (*Repository, error) {
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName), nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get respository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get repository failed, unable to get response body: %w", err)
	}

	repo := &Repository{}
	if err := json.Unmarshal([]byte(respBody), &repo); err != nil {
		return nil, fmt.Errorf("get repository failed, unable to unmarshall repository json, %w", err)
	}

	repo.setSessionInfo(resp)
	return repo, nil
}

func (p *Repositories) Delete(ctx context.Context, projectName, repoName string) error {
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repoName), nil, nil, &httpclient.Delete, nil)
	if err != nil {
		return fmt.Errorf("delete repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("delete repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (p *Repositories) Create(ctx context.Context, projectName string, repo *Repository) (*Repository, error) {
	hdr := http.Header{"Content-Type": []string{"application/json"}}

	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI), nil, repo, &httpclient.Post, &hdr)
	if err != nil {
		if resp.StatusCode == http.StatusConflict {
			return nil, gitprovider.ErrAlreadyExists
		}
		return nil, fmt.Errorf("create respository failed, %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("create repository failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), repo); err != nil {
		return nil, fmt.Errorf("create repository failed, unable to unmarshall repository json, %w", err)
	}

	repo.setSessionInfo(resp)

	return repo, nil
}

func (p *Repositories) Update(ctx context.Context, projectName string, repo *Repository) (*Repository, error) {
	hdr := http.Header{"Content-Type": []string{"application/json"}}

	resp, err := p.Requester.Do(ctx, newURI(projectsURI, projectName, RepositoriesURI, repo.Name), nil, repo, &httpclient.Put, &hdr)
	if err != nil {
		return nil, fmt.Errorf("update repository failed, %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("update repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("update repsository failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), repo); err != nil {
		return nil, fmt.Errorf("update repsository failed, unable to unmarshall repository json, %w", err)
	}

	repo.setSessionInfo(resp)

	return repo, nil
}

type RepositoryGroupPermission struct {
	SessionInfo `json:"sessionInfo,omitempty"`
	Group       struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func NewStashRepositoryGroups(client *stashClientImpl, projectName, repositoryName string) StashRepositoryGroups {
	p := &RepositoryGroups{
		Paging:         Paging{},
		Requester:      client.Client(),
		ProjectName:    projectName,
		RepositoryName: repositoryName,
		Groups:         make([]*RepositoryGroupPermission, 0),
		log:            client.log,
	}
	return p
}

type StashRepositoryGroups interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, groupName string) (*RepositoryGroupPermission, error)
	Create(ctx context.Context, permission *RepositoryGroupPermission) error
	getGroups() []*RepositoryGroupPermission
}

type RepositoryGroups struct {
	StashRepositoryGroups `json:"-"`
	Paging
	httpclient.Requester
	ProjectName    string                       `json:"-"`
	RepositoryName string                       `json:"-"`
	Groups         []*RepositoryGroupPermission `json:"values,omitempty"`
	log            logr.Logger
}

func (p *RepositoryGroups) getGroups() []*RepositoryGroupPermission {
	return p.Groups
}

func (p *RepositoryGroups) Create(ctx context.Context, permission *RepositoryGroupPermission) error {
	var query *url.Values = &url.Values{
		"name":       []string{permission.Group.Name},
		"permission": []string{permission.Permission},
	}
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, p.ProjectName, RepositoriesURI, p.RepositoryName, groupPermisionsURI), query, nil, &httpclient.Put, nil)
	if err != nil {
		return fmt.Errorf("add group permissions to repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("add group permissions to repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (p *RepositoryGroups) Get(ctx context.Context, groupName string) (*RepositoryGroupPermission, error) {
	var query *url.Values = &url.Values{
		filterKey: []string{groupName},
	}
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, p.ProjectName, RepositoriesURI, p.RepositoryName, groupPermisionsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get group permissions for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get group permissions for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get group permissions for repository failed, unable to obtain response body, %w", err)
	}

	permissions := &RepositoryGroups{}
	if err := json.Unmarshal([]byte(respBody), &permissions); err != nil {
		return nil, fmt.Errorf("get group permissions for repository failed, unable to unmarshall repository json, %w", err)
	}

	for _, groupPerm := range permissions.Groups {
		if groupPerm.Group.Name == groupName {
			groupPerm.setSessionInfo(resp)
			return groupPerm, nil
		}
	}
	return nil, gitprovider.ErrNotFound
}

func (p *RepositoryGroups) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, p.ProjectName, RepositoriesURI, p.RepositoryName, groupPermisionsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list groups permissions for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list groups permissions for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list groups permissions for repository failed, unable to obtain response body, %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list groups permissions for repository failed, unable to unmarshall repository json, %w", err)
	}

	for _, groupPerm := range p.getGroups() {
		groupPerm.setSessionInfo(resp)
	}

	return &p.Paging, nil
}

type RepositoryUserPermission struct {
	SessionInfo `json:"sessionInfo,omitempty"`
	User        `json:"user,omitempty"`
	Permission  string `json:"permission,omitempty"`
}

func NewStashRepositoryUsers(client stashClientImpl, repositoryName string) StashRepositoryUsers {
	p := &RepositoryUsers{
		Paging:         Paging{},
		Requester:      client.Client(),
		RepositoryName: repositoryName,
		Users:          make([]*RepositoryUserPermission, 0),
		log:            client.log,
	}
	return p
}

type StashRepositoryUsers interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	//GetMembers(ctx context.Context, repositoryName string) (*Repository, error)
	getUsers() []*RepositoryUserPermission
}

type RepositoryUsers struct {
	StashRepositoryUsers `json:"-"`
	Paging
	httpclient.Requester
	ProjectName    string                      `json:"-"`
	RepositoryName string                      `json:"-"`
	Users          []*RepositoryUserPermission `json:"values,omitempty"`
	log            logr.Logger
}

func (p *RepositoryUsers) getUsers() []*RepositoryUserPermission {
	return p.Users
}

func (p RepositoryUsers) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, p.ProjectName, userPermisionsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list users permissions for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list users permissions for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list users permissions for repository failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list users permissions for repository failed, unable to unmarshall json, %w", err)
	}

	for _, userPerm := range p.getUsers() {
		userPerm.setSessionInfo(resp)
	}

	return &p.Paging, nil
}
