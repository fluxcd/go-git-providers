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
	"strconv"
	"strings"
)

const (
	usersURI = "users"
)

var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = fmt.Errorf("the requested resource was not found")
)

// Users interface defines the methods that can be used to
// retrieve users.
type Users interface {
	List(ctx context.Context, opts *PagingOptions) (*UserList, error)
	Get(ctx context.Context, userName string) (*User, error)
}

// UsersService is a client for communicating with stash users endpoint
// Stash API docs: https://docs.atlassian.com/DAC/rest/stash/3.11.3/stash-rest.html
type UsersService service

// User represents a Stash user.
type User struct {
	// Session is the session information for the user.
	Session Session `json:"sessionInfo,omitempty"`
	// Active is true if the user is active.
	Active bool `json:"active,omitempty"`
	// Deletable is true if the user is deletable.
	Deletable bool `json:"deletable,omitempty"`
	// DirectoryName is the directory name where the user is saved.
	DirectoryName string `json:"directoryName,omitempty"`
	// DisplayName is the display name of the user.
	DisplayName string `json:"displayName,omitempty"`
	// EmailAddress is the email address of the user.
	EmailAddress string `json:"emailAddress,omitempty"`
	// ID is the unique identifier of the user.
	ID int64 `json:"id,omitempty"`
	// LastAuthenticationTimestamp is the last authentication timestamp of the user.
	LastAuthenticationTimestamp int64 `json:"lastAuthenticationTimestamp,omitempty"`
	// Links is the links to other resources.
	Links `json:"links,omitempty"`
	// MutableDetails is true if the user is mutable.
	MutableDetails bool `json:"mutableDetails,omitempty"`
	// MutableGroups is true if the groups are mutable.
	MutableGroups bool `json:"mutableGroups,omitempty"`
	// Name is the name of the user.
	Name string `json:"name,omitempty"`
	// Slug is the slug of the user.
	Slug string `json:"slug,omitempty"`
	// Type is the type of the user.
	Type string `json:"type,omitempty"`
}

// UserList is a list of users.
type UserList struct {
	// Paging is the paging information.
	Paging
	// Users is the list of Stash Users.
	Users []*User `json:"values,omitempty"`
}

// GetUsers retrieves a list of users.
func (u *UserList) GetUsers() []*User {
	return u.Users
}

// List retrieves a list of users.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a paging struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/api/1.0/users".
func (s *UsersService) List(ctx context.Context, opts *PagingOptions) (*UserList, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(usersURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list users request creation failed, %w", err)
	}

	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list users failed, %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	u := &UserList{
		Users: []*User{},
	}
	if err := json.Unmarshal(res, u); err != nil {
		return nil, fmt.Errorf("list users failed, unable to unmarshal repository list json, %w", err)
	}

	for _, r := range u.GetUsers() {
		r.Session.set(resp)
	}

	return u, nil
}

// Get retrieves a user by name.
// Get uses the endpoint "GET /rest/api/1.0/users/{userSlug}".
func (s *UsersService) Get(ctx context.Context, userSlug string) (*User, error) {
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newURI(usersURI, userSlug))
	if err != nil {
		return nil, fmt.Errorf("get user request creation failed, %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get user failed, %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	var user User

	if err := json.Unmarshal(res, &user); err != nil {
		return nil, fmt.Errorf("get user failed, unable to unmarshal repository list json, %w", err)
	}

	user.Session.set(resp)
	return &user, nil

}

// addPaging adds paging elements to URI query
func addPaging(query url.Values, opts *PagingOptions) url.Values {
	if query == nil {
		query = url.Values{}
	}

	if opts == nil {
		return query
	}

	if opts.Limit != 0 {
		query.Add("limit", strconv.Itoa(int(opts.Limit)))
	}

	if opts.Start != 0 {
		query.Add("start", strconv.Itoa(int(opts.Start)))
	}
	return query
}

// newURI builds stash URI
func newURI(elements ...string) string {
	return strings.Join(append([]string{stashURIprefix}, elements...), "/")
}
