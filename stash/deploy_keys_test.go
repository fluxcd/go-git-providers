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
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetKey(t *testing.T) {
	tests := []struct {
		name  string
		keyID int
	}{
		{
			name:  "test an access key",
			keyID: 1,
		},
		{
			name:  "test access key does not exist",
			keyID: -1,
		},
	}

	validKeyIDs := []string{"1"}

	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s/%s", stashURIkeys, projectsURI, RepositoriesURI, deployKeysURI, strconv.Itoa(tt.keyID))
			mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				for _, substr := range validKeyIDs {
					if path.Base(r.URL.String()) == substr {
						w.WriteHeader(http.StatusOK)
						k := &DeployKey{
							Key: Key{
								ID:    tt.keyID,
								Text:  "ssh-rsa AAAAB3... me@127.0.0.1",
								Label: "me@127.0.0.1",
							},
							Permission: "REPO_READ",
							Repository: Repository{
								Slug: "repo1",
								Name: "repository 1",
								Project: Project{
									Key:  "prj1",
									Name: "prjoject 1",
								},
							},
						}

						json.NewEncoder(w).Encode(k)
						return
					}
				}

				http.Error(w, "The specified access key does not exist", http.StatusNotFound)

				return

			})

			ctx := context.Background()
			k, err := client.DeployKeys.Get(ctx, "prj1", "repo1", tt.keyID)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("DeployKeys.Get returned error: %v", err)
				}
				return
			}

			if k.Key.ID != tt.keyID {
				t.Fatalf("DeployKeys.Get returned:\n%d, want:\n%d", k.Key.ID, tt.keyID)
			}

		})
	}
}

func TestListKeys(t *testing.T) {
	keyIDs := []*DeployKey{
		{Key: Key{ID: 1}},
		{Key: Key{ID: 2}},
		{Key: Key{ID: 2}},
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s", stashURIkeys, projectsURI, RepositoriesURI, deployKeysURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		k := struct {
			Values []*DeployKey `json:"values"`
		}{[]*DeployKey{
			keyIDs[0],
			keyIDs[1],
			keyIDs[2],
		},
		}
		json.NewEncoder(w).Encode(k)
		return

	})
	ctx := context.Background()
	list, err := client.DeployKeys.List(ctx, "prj1", "repo1", nil)
	if err != nil {
		t.Fatalf("DeployKeys..List returned error: %v", err)
	}

	if diff := cmp.Diff(keyIDs, list.DeployKeys); diff != "" {
		t.Errorf("DeployKeys..List returned diff (want -> got):\n%s", diff)
	}
}

func TestCreateKey(t *testing.T) {
	tests := []struct {
		name string
		key  DeployKey
	}{
		{
			name: "invalid key",
			key: DeployKey{
				Key: Key{
					Text: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABABABAQCBejb9KVtNtpxsBf3rWd6hbJD2VRTzR5g8/84mQafhRk1NoDa3cCYujl1PZXWnh+mCJwa6sf2VLD6oWc4ce9cjbWORxj3w/WUmcjykz8JlmqY0nLCV6oy8m7OeM4NYd/+Zwbz5DmwlXADGsDbJH9ndiCADOqsGXkkG9Mid+1oYYp2IsL/60ShHa1/pLcHjwVndL9Kf7gBMu5ezH0liXvUqX0C0MKevVvgYTG6bEpPLsN15EIGQKU38ogUuOsbcv6xoIHI1J3rL1nzQtMef0VeI4Jl4jQzT7sQuTzjjlLAUjmLRx2ub2cuaweWK30Ahvm5CRbmD8X0KTDK6FVm9oZH/",
				},
				Permission: "",
				Repository: Repository{
					Slug: "my-repo",
					Project: Project{
						Key: "prj",
					},
				},
			},
		},
		{
			name: "writer key",
			key: DeployKey{
				Key: Key{
					Text: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCBejb9KVtNtpxsBf3rWd6hbJD2VRTzR5g8/84mQafhRk1NoDa3cCYujl1PZXWnh+mCJwa6sf2VLD6oWc4ce9cjbWORxj3w/WUmcjykz8JlmqY0nLCV6oy8m7OeM4NYd/+Zwbz5DmwlXADGsDbJH9ndiCADOqsGXkkG9Mid+1oYYp2IsL/60ShHa1/pLcHjwVndL9Kf7gBMu5ezH0liXvUqX0C0MKevVvgYTG6bEpPLsN15EIGQKU38ogUuOsbcv6xoIHI1J3rL1nzQtMef0VeI4Jl4jQzT7sQuTzjjlLAUjmLRx2ub2cuaweWK30Ahvm5CRbmD8X0KTDK6FVm9oZH/",
				},
				Permission: "REPO_WRITE",
				Repository: Repository{
					Slug: "my-repo",
					Project: Project{
						Key: "prj",
					},
				},
			},
		},
		{
			name: "reader key",
			key: DeployKey{
				Key: Key{
					Text: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABABABAQCBejb9KVtNtpxsBf3rWd6hbJD2VRTzR5g8/84mQafhRk1NoDa3cCYujl1PZXWnh+mCJwa6sf2VLD6oWc4ce9cjbWORxj3w/WUmcjykz8JlmqY0nLCV6oy8m7OeM4NYd/+Zwbz5DmwlXADGsDbJH9ndiCADOqsGXkkG9Mid+1oYYp2IsL/60ShHa1/pLcHjwVndL9Kf7gBMu5ezH0liXvUqX0C0MKevVvgYTG6bEpPLsN15EIGQKU38ogUuOsbcv6xoIHI1J3rL1nzQtMef0VeI4Jl4jQzT7sQuTzjjlLAUjmLRx2ub2cuaweWK30Ahvm5CRbmD8X0KTDK6FVm9oZH/",
				},
				Permission: "REPO_READ",
				Repository: Repository{
					Slug: "my-repo",
					Project: Project{
						Key: "prj",
					},
				},
			},
		},
	}

	permissions := []string{"REPO_READ", "REPO_WRITE", "REPO_ADMIN"}
	IDCounter := 0

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj/%s/my-repo/%s", stashURIkeys, projectsURI, RepositoriesURI, deployKeysURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			req := &DeployKey{}
			json.NewDecoder(r.Body).Decode(req)
			for _, p := range permissions {
				if req.Permission == p {
					k := &DeployKey{
						Key: Key{
							ID:   IDCounter + 1,
							Text: req.Key.Text,
						},
						Permission: p,
						Repository: Repository{
							Slug: "my-repo",
							Project: Project{
								Key: "prj",
							},
						},
					}
					json.NewEncoder(w).Encode(k)
					return
				}
			}
		}

		http.Error(w, "The access key was not created due to wrong permission.", http.StatusBadRequest)

		return

	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			k, err := client.DeployKeys.Create(ctx, &tt.key)
			if err != nil {
				if !strings.Contains(err.Error(), strconv.Itoa(http.StatusBadRequest)) {
					t.Fatalf("DeployKeys.Create returned error: %v", err)
				}
				return
			}

			if (k.Key.Text != tt.key.Key.Text) || (k.Permission != tt.key.Permission) || (k.Repository.Slug != tt.key.Repository.Slug) {
				t.Errorf("DeployKeys.Create returned:\n%v, want:\n%v", k, tt.key)
			}
		})
	}
}

func TestUpdateKeyPermission(t *testing.T) {
	tests := []struct {
		name       string
		key        int
		permission string
	}{
		{
			name:       "test update to Writer",
			key:        1,
			permission: "REPO_WRITE",
		},
		{
			name:       "test update to Admin",
			key:        1,
			permission: "REPO_ADMIN",
		},
		{
			name:       "test update to REPO_BDFL",
			key:        1,
			permission: "REPO_REPO_BDFL",
		},
	}

	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s/%s/%s/%s", stashURIkeys, projectsURI, RepositoriesURI, deployKeysURI, strconv.Itoa(tt.key), keyPermisionsURI, tt.permission)
			mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut {
					req := &DeployKey{}
					json.NewDecoder(r.Body).Decode(req)
					switch path.Base(r.URL.String()) {
					case "REPO_WRITE":
						k := &DeployKey{
							Key: Key{
								ID:   1,
								Text: req.Key.Text,
							},
							Permission: "REPO_WRITE",
							Repository: Repository{
								Slug: "repo1",
								Project: Project{
									Key: "prj1",
								},
							},
						}
						json.NewEncoder(w).Encode(k)
						return
					case "REPO_ADMIN":
						k := &DeployKey{
							Key: Key{
								ID:   1,
								Text: req.Key.Text,
							},
							Permission: "REPO_ADMIN",
							Repository: Repository{
								Slug: "repo1",
								Project: Project{
									Key: "prj1",
								},
							},
						}
						json.NewEncoder(w).Encode(k)
						return
					default:
					}
				}

				http.Error(w, "The specified access key does not exist", http.StatusNotFound)

				return

			})
			ctx := context.Background()
			k, err := client.DeployKeys.UpdateKeyPermission(ctx, "prj1", "repo1", tt.key, tt.permission)
			if err != nil {
				if !strings.Contains(err.Error(), ErrNotFound.Error()) {
					t.Fatalf("UpdateRepositoryGroupPermission returned error: %v", err)
				}
				return
			}

			if k.Permission != tt.permission {
				t.Errorf("UpdateRepositoryGroupPermission returned:\n%v, want:\n%v", k, tt.permission)
			}
		})
	}
}
