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

	"errors"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
	"github.com/go-logr/logr"
)

var (
	ErrorGetGroupMultipleItems = errors.New("multiple items returned for group name")
)

const (
	groupsURI       = "admin/groups"
	groupMembersURI = "admin/groups/more-members"
)

// Group represents a stash group.
type Group struct {
	SessionInfo `json:"sessionInfo,omitempty"`
	Name        string `json:"name,omitempty"`
	Deleteable  bool   `json:"deletable,omitempty"`
}

func NewStashGroups(client *stashClientImpl) StashGroups {
	g := &Groups{
		Paging:    Paging{},
		Requester: client.Client(),
		Groups:    make([]*Group, 0),
		log:       client.log,
	}
	return g
}

type StashGroups interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, groupName string) (*Group, error)
	getGroups() []*Group
}

type Groups struct {
	StashGroups
	Paging
	httpclient.Requester
	Groups []*Group `json:"values,omitempty"`
	log    logr.Logger
}

func (g *Groups) getGroups() []*Group {
	return g.Groups
}

func (g *Groups) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := g.Requester.Do(ctx, newURI(groupsURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list groups failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list groups api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list groups failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &g); err != nil {
		return nil, fmt.Errorf("list groups failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range g.getGroups() {
		r.setSessionInfo(resp)
	}

	return &g.Paging, nil
}

func (g *Groups) Get(ctx context.Context, groupName string) (*Group, error) {
	var query *url.Values
	query = &url.Values{
		"filter": []string{groupName},
	}
	resp, err := g.Requester.Do(ctx, groupsURI, query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get group failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get group api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get group failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &g); err != nil {
		return nil, fmt.Errorf("get group failed, unable to unmarshal repository list json, %w", err)
	}

	for _, group := range g.getGroups() {
		if group.Name == groupName {
			group.setSessionInfo(resp)
			return group, nil
		}
	}

	return nil, gitprovider.ErrNotFound
}

func NewStashGroupMembers(client *stashClientImpl) StashGroupMembers {
	m := &GroupMembers{
		Paging:    Paging{},
		Requester: client.Client(),
		Users:     make([]*User, 0),
		log:       client.log,
	}
	return m
}

type StashGroupMembers interface {
	List(ctx context.Context, groupName string, opts *ListOptions) (*Paging, error)
	getGroupMembers() []*User
}

type GroupMembers struct {
	StashGroupMembers
	Paging
	httpclient.Requester
	GroupName string  `json:"-"`
	Users     []*User `json:"values,omitempty"`
	log       logr.Logger
}

func (m *GroupMembers) getGroupMembers() []*User {
	return m.Users
}

func (m *GroupMembers) List(ctx context.Context, groupName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	query = setKeyValues(query, contextKey, groupName)
	resp, err := m.Requester.Do(ctx, newURI(groupMembersURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list group members failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list group members api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list group members failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &m); err != nil {
		return nil, fmt.Errorf("list group members failed, unable to unmarshal repository list json, %w", err)
	}

	for _, member := range m.getGroupMembers() {
		member.setSessionInfo(resp)
	}

	return &m.Paging, nil
}
