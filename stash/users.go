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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
	"github.com/go-logr/logr"
)

const (
	usersURI = "users"
)

type User struct {
	SessionInfo                 `json:"sessionInfo,omitempty"`
	Active                      bool   `json:"active,omitempty"`
	Deletable                   bool   `json:"deletable,omitempty"`
	DirectoryName               string `json:"directoryName,omitempty"`
	DisplayName                 string `json:"displayName,omitempty"`
	EmailAddress                string `json:"emailAddress,omitempty"`
	ID                          int64  `json:"id,omitempty"`
	LastAuthenticationTimestamp int64  `json:"lastAuthenticationTimestamp,omitempty"`
	Links                       `json:"links,omitempty"`
	MutableDetails              bool   `json:"mutableDetails,omitempty"`
	MutableGroups               bool   `json:"mutableGroups,omitempty"`
	Name                        string `json:"name,omitempty"`
	Slug                        string `json:"slug,omitempty"`
	Type                        string `json:"type,omitempty"`
}

func NewStashUsers(client *stashClientImpl) StashUsers {
	g := &Users{
		Paging:    Paging{},
		Requester: client.Client(),
		Users:     make([]*User, 0),
		log:       client.log,
	}
	return g
}

type StashUsers interface {
	List(ctx context.Context, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, userName string) (*User, error)
	geUsers() []*User
}

type Users struct {
	StashUsers
	Paging
	httpclient.Requester
	Users []*User `json:"values,omitempty"`
	log   logr.Logger
}

func (g *Users) getUsers() []*User {
	return g.Users
}

func (g *Users) List(ctx context.Context, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := g.Requester.Do(ctx, newURI(usersURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list users failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list users api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list users failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &g); err != nil {
		return nil, fmt.Errorf("list users failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range g.getUsers() {
		r.setSessionInfo(resp)
	}

	return &g.Paging, nil
}

func (g *Users) Get(ctx context.Context, userName string) (*User, error) {
	var query *url.Values
	query = addPaging(query, &ListOptions{})
	resp, err := g.Requester.Do(ctx, newURI(usersURI), nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get user failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get user failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &g); err != nil {
		return nil, fmt.Errorf("get user failed, unable to unmarshal repository list json, %w", err)
	}

	for _, user := range g.getUsers() {
		if user.Name == userName {
			user.setSessionInfo(resp)
			return user, nil
		}
	}

	return nil, gitprovider.ErrNotFound
}
