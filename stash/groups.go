/*
Copyright 2021 The Flux authors

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
)

const (
	groupsURI       = "admin/groups"
	groupMembersURI = "admin/groups/more-members"
)

// Groups interface defines the methods that can be used to
// retrieve groups and members of a group.
type Groups interface {
	List(ctx context.Context, opts *PagingOptions) (*GroupList, error)
	Get(ctx context.Context, groupName string) (*Group, error)
	ListGroupMembers(ctx context.Context, groupName string, opts *PagingOptions) (*GroupMembers, error)
	AllGroupMembers(ctx context.Context, groupName string) ([]*User, error)
}

// GroupsService is a client for communicating with stash groups endpoint
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
type GroupsService service

// Group represents a stash group.
type Group struct {
	// Session is the session object for the group.
	Session Session `json:"sessionInfo,omitempty"`
	// Name is the name of the group.
	Name string `json:"name,omitempty"`
	// Delete is the delete flag for the group.
	Deleteable bool `json:"deletable,omitempty"`
}

// GroupList represents  a list of stash groups.
type GroupList struct {
	// Paging is the paging information.
	Paging
	// Groups is the list of stash groups.
	Groups []*Group `json:"values,omitempty"`
}

// GetGroups returns a slice of groups.
func (g *GroupList) GetGroups() []*Group {
	return g.Groups
}

// List retrieves a list of stash groups.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a paging struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/admin/groups".
// The authenticated user must have the LICENSED_USER permission to call this resource.
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *GroupsService) List(ctx context.Context, opts *PagingOptions) (*GroupList, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(groupsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list groups failed: , %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list groups failed: , %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	g := &GroupList{
		Groups: []*Group{},
	}
	if err := json.Unmarshal(res, g); err != nil {
		return nil, fmt.Errorf("list groups failed, unable to unmarshal repository list json: , %w", err)
	}

	for _, r := range g.GetGroups() {
		r.Session.set(resp)
	}

	return g, nil
}

// Get retrieves a stash group given it's name.
// Get uses the endpoint "GET /rest/api/1.0/admin/groups".
// The authenticated user must have the LICENSED_USER permission to call this resource.
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *GroupsService) Get(ctx context.Context, groupName string) (*Group, error) {
	query := url.Values{
		"filter": []string{groupName},
	}

	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(groupsURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list groups failed: , %w", err)
	}

	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get group failed: , %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	g := &Group{
		Name: groupName,
	}

	if err := json.Unmarshal(res, g); err != nil {
		return nil, fmt.Errorf("get group failed, unable to unmarshal repository list json: , %w", err)
	}

	g.Session.set(resp)
	return g, nil
}

// GroupMembers  is a list of stash groups members.
type GroupMembers struct {
	// Paging is the paging information.
	Paging
	// GroupName is the name of the group.
	GroupName string `json:"-"`
	// Users is the list of stash groups members.
	Users []*User `json:"values,omitempty"`
}

// GetGroupMembers retrieves a list of stash groups members.
func (m *GroupMembers) GetGroupMembers() []*User {
	return m.Users
}

// ListGroupMembers retrieves a list of stash groups members.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a paging struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/admin/groups/more-members".
// The authenticated user must have the LICENSED_USER permission to call this resource.
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-rest.html
func (s *GroupsService) ListGroupMembers(ctx context.Context, groupName string, opts *PagingOptions) (*GroupMembers, error) {
	query := addPaging(url.Values{}, opts)
	// The group name is required as a parameter
	query.Set(contextKey, groupName)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(groupMembersURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list groups request creation failed: , %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list group members failed: , %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	m := &GroupMembers{
		GroupName: groupName,
		Users:     []*User{},
	}

	if err := json.Unmarshal(res, m); err != nil {
		return nil, fmt.Errorf("list group members failed, unable to unmarshal repository list json: , %w", err)
	}

	for _, member := range m.GetGroupMembers() {
		member.Session.set(resp)
	}

	return m, nil
}

// AllGroupMembers retrieves all group members.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *GroupsService) AllGroupMembers(ctx context.Context, groupName string) ([]*User, error) {
	p := []*User{}
	opts := &PagingOptions{Limit: perPageLimit}
	err := allPages(opts, func() (*Paging, error) {
		list, err := s.ListGroupMembers(ctx, groupName, opts)
		if err != nil {
			return nil, err
		}
		p = append(p, list.GetGroupMembers()...)
		return &list.Paging, nil
	})
	if err != nil {
		return nil, err
	}

	return p, nil
}
