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

	"github.com/pkg/errors"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
)

var (
	ErrorGetProjectMultipleItems = errors.New("multiple items returned for group name")
)

const (
	projectsURI        = "projects"
	groupPermisionsURI = "permissions/groups"
	userPermisionsURI  = "permissions/users"
)

type Project struct {
	Description string `json:"description,omitempty"`
	ID          int64  `json:"id,omitempty"`
	Key         string `json:"key,omitempty"`
	Links       `json:"links,omitempty"`
	User        `json:"owner,omitempty"`
	Name        string `json:"name,omitempty"`
	Public      bool   `json:"public,omitempty"`
	Type        string `json:"type,omitempty"`
}

func NewStashProjects(client httpclient.ReqResp) StashProjects {
	p := &Projects{
		Paging:   Paging{},
		ReqResp:  client,
		Projects: make([]*Project, 0),
	}
	return p
}

type StashProjects interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName string) (*Project, error)
	getProjects() []*Project
}

type Projects struct {
	StashProjects
	Paging
	httpclient.ReqResp
	Projects []*Project `json:"values,omitempty"`
}

func (p *Projects) getProjects() []*Project {
	return p.Projects
}

func (p *Projects) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI), query, nil, nil, nil)
	if err != nil {
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list projects\n")

	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal project list\n")
	}

	return &p.Paging, nil
}

func (p *Projects) Get(ctx context.Context, projectName string) (*Project, error) {
	var query *url.Values = nil
	query = &url.Values{
		"name": []string{projectName},
	}
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI), query, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to get project\n")
	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshall project json\n")
	}
	for _, project := range p.Projects {
		if project.Name == projectName {
			return project, nil
		}
	}
	return nil, gitprovider.ErrNotFound
}

func NewStashProjectRepos(client httpclient.ReqResp) StashProjectRepos {
	p := &ProjectRepos{
		Paging:  Paging{},
		ReqResp: client,
		Repos:   make([]*Repository, 0),
	}
	return p
}

type StashProjectRepos interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName string) (*Project, error)
	getRepos() []*Repository
}

type ProjectRepos struct {
	StashProjectRepos
	Paging
	httpclient.ReqResp
	ProjectName string        `json:"-"`
	Repos       []*Repository `json:"values,omitempty"`
}

func (p *ProjectRepos) getRepos() []*Repository {
	return p.Repos
}

func (p *ProjectRepos) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, p.ProjectName, RepositoriesURI), query, nil, nil, nil)
	if err != nil {
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list project repos\n")

	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal project repo list\n")
	}

	return &p.Paging, nil
}

type ProjectGroupPermission struct {
	Group struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func NewStashProjectGroups(client httpclient.ReqResp, projectName string) StashProjectGroups {
	p := &ProjectGroups{
		Paging:      Paging{},
		ReqResp:     client,
		ProjectName: projectName,
		Groups:      make([]*ProjectGroupPermission, 0),
	}
	return p
}

type StashProjectGroups interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	//GetMembers(ctx context.Context, projectName string) (*Project, error)
	getGroups() []*ProjectGroupPermission
}

type ProjectGroups struct {
	StashProjects
	Paging
	httpclient.ReqResp
	ProjectName string                    `json:"-"`
	Groups      []*ProjectGroupPermission `json:"values,omitempty"`
}

func (p *ProjectGroups) getGroups() []*ProjectGroupPermission {
	return p.Groups
}

func (p *ProjectGroups) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, p.ProjectName, groupPermisionsURI), query, nil, nil, nil)
	if err != nil {
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list projects\n")

	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal project list\n")
	}

	return &p.Paging, nil
}

type ProjectUserPermission struct {
	User       User   `json:"user,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func NewStashProjectUsers(client httpclient.ReqResp, projectName string) StashProjectUsers {
	p := &ProjectUsers{
		Paging:      Paging{},
		ReqResp:     client,
		ProjectName: projectName,
		Users:       make([]*ProjectUserPermission, 0),
	}
	return p
}

type StashProjectUsers interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	//GetMembers(ctx context.Context, projectName string) (*Project, error)
	getUsers() []*ProjectUserPermission
}

type ProjectUsers struct {
	StashProjects
	Paging
	httpclient.ReqResp
	ProjectName string                   `json:"-"`
	Users       []*ProjectUserPermission `json:"values,omitempty"`
}

func (p *ProjectUsers) getUsers() []*ProjectUserPermission {
	return p.Users
}

func (p ProjectUsers) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	resp, err := p.ReqResp.Do(ctx, newURI(projectsURI, p.ProjectName, userPermisionsURI), query, nil, nil, nil)
	if err != nil {
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list projects\n")

	}

	if err := json.Unmarshal([]byte(resp), &p); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal project list\n")
	}

	return &p.Paging, nil
}
