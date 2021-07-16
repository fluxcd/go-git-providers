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

	"github.com/fluxcd/go-git-providers/httpclient"
)

var (
	ErrorGetGroupMultipleItems = errors.New("multiple items returned for group name")
)

const (
	groupsURI       = "admin/groups"
	groupMembersURI = "admin/groups/more-members"
)

type User struct {
	Active                      bool   `json:"active,omitempty"`
	Deletable                   bool   `json:"deletable,omitempty"`
	DirectoryName               string `json:"directoryName,omitempty"`
	DisplayName                 string `json:"displayName,omitempty"`
	EmailAddress                string `json:"emailAddress,omitempty"`
	ID                          int64  `json:"id,omitempty"`
	LastAuthenticationTimestamp int64  `json:"lastAuthenticationTimestamp,omitempty"`
	Links
	MutableDetails bool   `json:"mutableDetails,omitempty"`
	MutableGroups  bool   `json:"mutableGroups,omitempty"`
	Name           string `json:"name,omitempty"`
	Slug           string `json:"slug,omitempty"`
	Type           string `json:"type,omitempty"`
}

/*
type Users struct {
	Paging
	Users []User `json:"values,omitempty"`
}
*/

// Group represents a stash group.
type Group struct {
	Name       string
	Deleteable bool
}

func NewStashGroups(client httpclient.ReqResp) StashGroups {
	g := &Groups{
		Paging:  Paging{},
		ReqResp: client,
		Groups:  make([]*Group, 0),
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
	httpclient.ReqResp
	Groups []*Group `json:"values,omitempty"`
}

func (g *Groups) getGroups() []*Group {
	return g.Groups
}

func (g *Groups) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	code, resp, err := g.ReqResp.Do(ctx, newURI(groupsURI), query, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list groups\n")
	}

	if err := json.Unmarshal([]byte(resp), &g); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal group list\n")
	}

	return &g.Paging, nil
}

func (g *Groups) Get(ctx context.Context, groupName string) (*Group, error) {
	var query *url.Values = nil
	query = &url.Values{
		"filter": []string{groupName},
	}
	code, resp, err := g.ReqResp.Do(ctx, groupsURI, query, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to get group\n")
	}

	if err := json.Unmarshal([]byte(resp), &g); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshall group json\n")
	}
	if len(g.Groups) != 1 {
		return nil, errors.Wrapf(ErrorGetGroupMultipleItems, "get project failed\n")
	}
	return g.Groups[0], nil
}

func NewStashGroupMembers(client httpclient.ReqResp) StashGroupMembers {
	m := &GroupMembers{
		Paging:  Paging{},
		ReqResp: client,
		Users:   make([]*User, 0),
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
	httpclient.ReqResp
	GroupName string  `json:"-"`
	Users     []*User `json:"values,omitempty"`
}

func (m *GroupMembers) getGroupMembers() []*User {
	return m.Users
}

func (m *GroupMembers) List(ctx context.Context, groupName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values = nil
	query = addPaging(query, opts)
	query = setKeyValues(query, contextKey, groupName)
	code, resp, err := m.ReqResp.Do(ctx, newURI(groupMembersURI), query, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, errors.WithMessagef(err, "failed to list group members\n")
	}

	if err := json.Unmarshal([]byte(resp), &m); err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal group member list\n")
	}

	return &m.Paging, nil
}
