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

	"github.com/go-logr/logr"

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
	SessionInfo `json:"sessionInfo,omitempty"`
	Description string `json:"description,omitempty"`
	ID          int64  `json:"id,omitempty"`
	Key         string `json:"key,omitempty"`
	Links       `json:"links,omitempty"`
	User        `json:"owner,omitempty"`
	Name        string `json:"name,omitempty"`
	Public      bool   `json:"public,omitempty"`
	Type        string `json:"type,omitempty"`
}

func NewStashProjects(client *stashClientImpl) StashProjects {
	p := &Projects{
		Paging:    Paging{},
		Requester: client.Client(),
		Projects:  make([]*Project, 0),
		log:       client.log,
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
	httpclient.Requester
	Projects []*Project `json:"values,omitempty"`
	log      logr.Logger
}

func (p *Projects) getProjects() []*Project {
	return p.Projects
}

func (p *Projects) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list projects failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list projects api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list projects failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list projects failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range p.getProjects() {
		r.setSessionInfo(resp)
	}

	return &p.Paging, nil
}

func (p *Projects) Get(ctx context.Context, projectName string) (*Project, error) {
	var query *url.Values
	query = &url.Values{
		"name": []string{projectName},
	}
	resp, err := p.Requester.Do(ctx, newURI(projectsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get project failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get project api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get project failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("get project failed, unable to unmarshal repository list json, %w", err)
	}
	for _, project := range p.Projects {
		if project.Name == projectName {
			return project, nil
		}
	}

	return nil, gitprovider.ErrNotFound
}

type ProjectGroupPermission struct {
	SessionInfo `json:"sessionInfo,omitempty"`
	Group       struct {
		Name string `json:"name,omitempty"`
	} `json:"group,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func NewStashProjectGroups(client *stashClientImpl, projectName string) StashProjectGroups {
	p := &ProjectGroups{
		Paging:      Paging{},
		Requester:   client.Client(),
		ProjectName: projectName,
		Groups:      make([]*ProjectGroupPermission, 0),
		log:         client.log,
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
	httpclient.Requester
	ProjectName string                    `json:"-"`
	Groups      []*ProjectGroupPermission `json:"values,omitempty"`
	log         logr.Logger
}

func (p *ProjectGroups) getGroups() []*ProjectGroupPermission {
	return p.Groups
}

func (p *ProjectGroups) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, p.ProjectName, groupPermisionsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list project groups failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list project groups api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list project groups failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list project groups failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range p.getGroups() {
		r.setSessionInfo(resp)
	}

	return &p.Paging, nil
}

type ProjectUserPermission struct {
	SessionInfo `json:"sessionInfo,omitempty"`
	User        User   `json:"user,omitempty"`
	Permission  string `json:"permission,omitempty"`
}

func NewStashProjectUsers(client *stashClientImpl, projectName string) StashProjectUsers {
	p := &ProjectUsers{
		Paging:      Paging{},
		Requester:   client.Client(),
		ProjectName: projectName,
		Users:       make([]*ProjectUserPermission, 0),
		log:         client.log,
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
	httpclient.Requester
	ProjectName string                   `json:"-"`
	Users       []*ProjectUserPermission `json:"values,omitempty"`
	log         logr.Logger
}

func (p *ProjectUsers) getUsers() []*ProjectUserPermission {
	return p.Users
}

func (p ProjectUsers) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := p.Requester.Do(ctx, newURI(projectsURI, p.ProjectName, userPermisionsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list project users failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list project users api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list project users failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &p); err != nil {
		return nil, fmt.Errorf("list project users failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range p.getUsers() {
		r.setSessionInfo(resp)
	}

	return &p.Paging, nil
}
