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
	"strconv"
	"strings"

	"github.com/go-logr/logr"

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/httpclient"
)

const (
	deployKeysURI    = "ssh"
	keyPermisionsURI = "permission"
)

type Key struct {
	ID    int    `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Text  string `json:"text,omitempty"`
}

type DeployKey struct {
	SessionInfo `json:"sessionInfo,omitempty"`
	Key         `json:"key,omitempty"`
	Permission  string `json:"permission,omitempty"`
	Repository  `json:"repository,omitempty"`
}

func (d *DeployKey) CanPush() bool {
	return d.Permission != "REPO_READ"
}

func (d *DeployKey) ValidateInfo() error {
	if len(d.Permission) == 0 {
		return gitprovider.ErrInvalidPermissionLevel
	}
	if len(strings.Split(string(d.Key.Text), " ")) < 2 {
		return errors.New("invalid deploy key, expecting at least two fields, type and key")
	}
	return nil
}

func (d *DeployKey) Equals(actual gitprovider.InfoRequest) bool {
	return reflect.DeepEqual(d, actual)
}

func NewStashDeployKeys(client *stashClientImpl) StashDeployKeys {
	d := &DeployKeys{
		Paging:     Paging{},
		Requester:  client.Client(),
		DeployKeys: make([]*DeployKey, 0),
		log:        client.log,
	}
	return d
}

type StashDeployKeys interface {
	List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error)
	Get(ctx context.Context, projectName, repoName string, keyNum int) (*DeployKey, error)
	getDeployKeys() []*DeployKey
	Create(ctx context.Context, deployKey *DeployKey) (*DeployKey, error)
	Update(ctx context.Context, projectName, repoName string, keyNum int, permission string) (*DeployKey, error)
	Delete(ctx context.Context, projectName, repoName string, keyNum int) error
}

type DeployKeys struct {
	StashDeployKeys
	Paging
	httpclient.Requester
	DeployKeys []*DeployKey `json:"values,omitempty"`
	log        logr.Logger
}

func (d *DeployKeys) getDeployKeys() []*DeployKey {
	return d.DeployKeys
}

func (d *DeployKeys) List(ctx context.Context, projectName, repoName string, opts *ListOptions) (*Paging, error) {
	var query *url.Values
	query = addPaging(query, opts)
	resp, err := d.Requester.Do(ctx, newKeysURI(projectsURI, projectName, RepositoriesURI, repoName, deployKeysURI), query, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list deploy keys for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list deploy keys for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("list deploy keys for repository failed, unable to obtain response body: %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), &d); err != nil {
		return nil, fmt.Errorf("list deploy keys for repository failed, unable to unmarshall json, %w", err)
	}

	for _, deployKey := range d.getDeployKeys() {
		deployKey.setSessionInfo(resp)
	}

	return &d.Paging, nil
}

func (d *DeployKeys) Get(ctx context.Context, projectName, repoName string, keyNum int) (*DeployKey, error) {
	resp, err := d.Requester.Do(ctx, newKeysURI(projectsURI, projectName, RepositoriesURI, repoName, deployKeysURI, strconv.Itoa(keyNum)), nil, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get deploy key for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get deploy key for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("get deploy key for repository failed, unable to obtain response body, %w", err)
	}

	key := &DeployKey{}
	if err := json.Unmarshal([]byte(respBody), key); err != nil {
		return nil, fmt.Errorf("get deploy key for repository failed, unable to unmarshall repository json, %w", err)
	}

	return key, nil
}

func (d *DeployKeys) Delete(ctx context.Context, projectName, repoName string, keyNum int) error {
	resp, err := d.Requester.Do(ctx, newKeysURI(projectsURI, projectName, RepositoriesURI, repoName, deployKeysURI, strconv.Itoa(keyNum)), nil, nil, &httpclient.Delete, nil)
	if err != nil {
		return fmt.Errorf("delete deploy key for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete deploy key for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (d *DeployKeys) Create(ctx context.Context, deployKey *DeployKey) (*DeployKey, error) {
	hdr := http.Header{"Content-Type": []string{"application/json"}}

	resp, err := d.Requester.Do(ctx, newKeysURI(projectsURI, deployKey.Repository.Project.Key, RepositoriesURI, deployKey.Repository.Name, deployKeysURI), nil, deployKey, &httpclient.Post, &hdr)
	if err != nil {
		if resp.StatusCode == http.StatusConflict {
			return nil, gitprovider.ErrAlreadyExists
		}
		return nil, fmt.Errorf("create deploy key for repository failed, %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create deploy key for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("create deploy key for repository failed, unable to obtain response body, %w", err)
	}

	if err := json.Unmarshal([]byte(respBody), deployKey); err != nil {
		return nil, fmt.Errorf("create deploy key for repository failed, unable to unmarshall repository json, %w", err)
	}

	return deployKey, nil
}

func (d *DeployKeys) Update(ctx context.Context, projectName, repoName string, keyNum int, permission string) (*DeployKey, error) {
	hdr := http.Header{"Content-Type": []string{"application/json"}}

	resp, err := d.Requester.Do(ctx, newKeysURI(projectsURI, projectName, RepositoriesURI, repoName, deployKeysURI, strconv.Itoa(keyNum), keyPermisionsURI, permission), nil, nil, &httpclient.Put, &hdr)
	if err != nil {
		return nil, fmt.Errorf("update deploy key for repository failed, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update deploy key for repository api call returned unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := httpclient.GetRespBody(resp)
	if err != nil {
		return nil, fmt.Errorf("update deploy key for repository failed, unable to obtain response body, %w", err)
	}

	deployKey := &DeployKey{}
	if err := json.Unmarshal([]byte(respBody), deployKey); err != nil {
		return nil, fmt.Errorf("update deploy key for repository failed, unable to unmarshall repository json, %w", err)
	}

	return deployKey, nil
}
