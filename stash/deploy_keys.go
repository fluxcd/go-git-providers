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
	deployKeysURI    = "ssh"
	stashURIkeys     = "/rest/keys/1.0"
	keyPermisionsURI = "permission"
)

var (
	// ErrBadRequest is returned when a request is malformed.
	ErrBadRequest = fmt.Errorf("Bad request")
)

// DeployKeys interface defines the methods for working with
// repository access keys
type DeployKeys interface {
	List(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*DeployKeyList, error)
	All(ctx context.Context, projectKey, repositorySlug string) ([]*DeployKey, error)
	Get(ctx context.Context, projectKey, repositorySlug string, keyID int) (*DeployKey, error)
	Create(ctx context.Context, deployKey *DeployKey) (*DeployKey, error)
	Delete(ctx context.Context, projectKey, repositorySlug string, keyID int) error
	UpdateKeyPermission(ctx context.Context, projectKey, repositorySlug string, keyID int, permission string) (*DeployKey, error)
}

// DeployKeysService is a client for communicating with stash ssh keys endpoint
// bitbucket-server API docs: https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-ssh-rest.html
type DeployKeysService service

// Key is a ssh key
type Key struct {
	// ID is the key id
	ID int `json:"id,omitempty"`
	// Label is the key label
	Label string `json:"label,omitempty"`
	// Text is the key text
	// For example "text": "ssh-rsa AAAAB3... me@127.0.0.1"
	Text string `json:"text,omitempty"`
}

// DeployKey is an access key for a repository
type DeployKey struct {
	// Session is the session object
	Session `json:"sessionInfo,omitempty"`
	// Key is the key object
	Key `json:"key,omitempty"`
	// Permissions is the key permission
	// Available repository permissions are:
	// REPO_READ
	// REPO_WRITE
	// REPO_ADMIN
	Permission string `json:"permission,omitempty"`
	// Repository is the repository object
	Repository `json:"repository,omitempty"`
}

// DeployKeyList is a list of access keys
type DeployKeyList struct {
	Paging
	DeployKeys []*DeployKey `json:"values,omitempty"`
}

// GetDeployKeys returns the list of deploy keys
func (d *DeployKeyList) GetDeployKeys() []*DeployKey {
	return d.DeployKeys
}

// List returns the list of access keys for the repository.
// Paging is optional and is enabled by providing a PagingOptions struct.
// A pointer to a DeployKeyList struct is returned to retrieve the next page of results.
// List uses the endpoint "GET /rest/keys/1.0/projects/{projectKey}/repos/{repositorySlug}/ssh".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-ssh-rest.html
func (s *DeployKeysService) List(ctx context.Context, projectKey, repositorySlug string, opts *PagingOptions) (*DeployKeyList, error) {
	query := addPaging(url.Values{}, opts)
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newKeysURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, deployKeysURI), WithQuery(query))
	if err != nil {
		return nil, fmt.Errorf("list deploy keys for repository request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list deploy keys for repository requests failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	keys := &DeployKeyList{}
	if err := json.Unmarshal(res, keys); err != nil {
		return nil, fmt.Errorf("list deploy keys for repository failed, unable to unmarshall json: %w", err)
	}

	for _, k := range keys.GetDeployKeys() {
		k.Session.set(resp)
	}

	return keys, nil
}

// All retrieves all repository keys.
// This function handles pagination, HTTP error wrapping, and validates the server result.
func (s *DeployKeysService) All(ctx context.Context, projectKey, repositorySlug string) ([]*DeployKey, error) {
	k := []*DeployKey{}
	opts := &PagingOptions{Limit: perPageLimit}
	err := allPages(opts, func() (*Paging, error) {
		list, err := s.List(ctx, projectKey, repositorySlug, opts)
		if err != nil {
			return nil, err
		}
		k = append(k, list.GetDeployKeys()...)
		return &list.Paging, nil
	})
	if err != nil {
		return nil, err
	}

	return k, nil
}

// Get retrieves an access key given it's ID.
// Get uses the endpoint "GET /rest/keys/1.0/projects/{projectKey}/repos/{repositorySlug}/ssh/{keyId}".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-ssh-rest.html
func (s *DeployKeysService) Get(ctx context.Context, projectKey, repositorySlug string, keyID int) (*DeployKey, error) {
	req, err := s.Client.NewRequest(ctx, http.MethodGet, newKeysURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, deployKeysURI, strconv.Itoa(keyID)))
	if err != nil {
		return nil, fmt.Errorf("get deploy key for repository  request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get deploy key for repository requests failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	key := &DeployKey{}
	if err := json.Unmarshal(res, key); err != nil {
		return nil, fmt.Errorf("get deploy key for repository failed, unable to unmarshall repository json: %w", err)
	}

	key.Session.set(resp)

	return key, nil
}

// Create creates an access key.
// Create uses the endpoint "POST /rest/keys/1.0/projects/{projectKey}/repos/{repositorySlug}/ssh".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-ssh-rest.html
func (s *DeployKeysService) Create(ctx context.Context, deployKey *DeployKey) (*DeployKey, error) {
	header := http.Header{"Content-Type": []string{"application/json"}}
	body, err := marshallBody(deployKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall deploykey: %v", err)
	}
	req, err := s.Client.NewRequest(ctx, http.MethodPost, newKeysURI(projectsURI, deployKey.Repository.Project.Key, RepositoriesURI, deployKey.Repository.Slug, deployKeysURI), WithBody(body), WithHeader(header))
	if err != nil {
		return nil, fmt.Errorf("create deploy key for repository  request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create deploy key for repository requests failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("create deploy key for repository failed: %s", resp.Status)
	}

	key := &DeployKey{}
	if err := json.Unmarshal(res, key); err != nil {
		return nil, fmt.Errorf("create deploy key for repository failed, unable to unmarshall repository json: %w", err)
	}

	key.Session.set(resp)

	return key, nil
}

// Delete deletes the access key with the given ID
// Delete uses the endpoint "Delete /rest/keys/1.0/projects/{projectKey}/repos/{repositorySlug}/ssh/{keyId}".
// https://docs.atlassian.com/bitbucket-server/rest/5.16.0/bitbucket-ssh-rest.html
func (s *DeployKeysService) Delete(ctx context.Context, projectKey, repositorySlug string, keyID int) error {
	req, err := s.Client.NewRequest(ctx, http.MethodDelete, newKeysURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, deployKeysURI, strconv.Itoa(keyID)))
	if err != nil {
		return fmt.Errorf("delete deploy key for repository  request creation failed: %w", err)
	}
	_, _, err = s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("delete deploy key for repository requests failed: %w", err)
	}

	return nil
}

// UpdateKeyPermission updates the given access key permission
// UpdateKeyPermission uses the endpoint "PUT /rest/keys/1.0/projects/{projectKey}/ssh/{keyId}/permission/{permission}".
//
func (s *DeployKeysService) UpdateKeyPermission(ctx context.Context, projectKey, repositorySlug string, keyID int, permission string) (*DeployKey, error) {
	req, err := s.Client.NewRequest(ctx, http.MethodPut, newKeysURI(projectsURI, projectKey, RepositoriesURI, repositorySlug, deployKeysURI, strconv.Itoa(keyID),
		keyPermisionsURI, permission))
	if err != nil {
		return nil, fmt.Errorf("update deploy key permission for repository request creation failed: %w", err)
	}
	res, resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update deploy key permission for repository requests failed: %w", err)
	}

	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	key := &DeployKey{}
	if err := json.Unmarshal(res, key); err != nil {
		return nil, fmt.Errorf("update deploy key for repository failed, unable to unmarshall repository json: %w", err)
	}

	key.Session.set(resp)

	return key, nil
}

// newKeysURI builds stash keys URI
func newKeysURI(elements ...string) string {
	return strings.Join(append([]string{stashURIkeys}, elements...), "/")
}
