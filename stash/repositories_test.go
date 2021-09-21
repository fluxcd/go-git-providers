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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetRepository(t *testing.T) {
	tests := []struct {
		name           string
		projectKey     string
		repositorySlug string
	}{
		{
			name:           "test repository does not exist",
			projectKey:     "prj",
			repositorySlug: "default",
		},
		{
			name:           "test a repository",
			projectKey:     "prj",
			repositorySlug: "repo1",
		},
	}

	validNames := []string{"repo1"}
	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}".
			p := fmt.Sprintf("%s/%s/%s/%s/%s", stashURIprefix, projectsURI, tt.projectKey, RepositoriesURI, tt.repositorySlug)
			mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				for _, substr := range validNames {
					if path.Base(r.URL.String()) == substr {
						w.WriteHeader(http.StatusOK)
						u := &Repository{
							Slug: tt.repositorySlug,
						}
						json.NewEncoder(w).Encode(u)
						return
					}
				}

				http.Error(w, "The specified repository does not exist", http.StatusNotFound)

				return

			})
			ctx := context.Background()
			r, err := client.Repositories.Get(ctx, tt.projectKey, tt.repositorySlug)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("Repositories.Get returned error: %v", err)
				}
				return
			}

			if diff := cmp.Diff(&Repository{Slug: tt.repositorySlug}, r); diff != "" {
				t.Fatalf("Repositories.Get returned diff (want -> got):\n%s", diff)
			}

		})
	}
}

func TestListRepositories(t *testing.T) {
	tests := []struct {
		name       string
		projectKey string
		repos      []*Repository
	}{
		{
			name:       "test a list of repositories",
			projectKey: "prj1",
			repos: []*Repository{
				{
					Slug: "repo1",
				},
				{
					Slug: "repo2",
				},
			},
		},
		{
			name:       "test a list of user repositories",
			projectKey: "~johnsmith",
			repos: []*Repository{
				{
					Slug: "userRepo1",
				},
				{
					Slug: "userRepo2",
				},
			},
		},
	}

	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//http: //example.com/rest/api/1.0/projects/~johnsmith/repos
			path := fmt.Sprintf("%s/%s/%s/%s", stashURIprefix, projectsURI, tt.projectKey, RepositoriesURI)

			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				u := struct {
					Repositores []*Repository `json:"values"`
				}{
					Repositores: tt.repos,
				}
				json.NewEncoder(w).Encode(u)
				return

			})
			ctx := context.Background()
			list, err := client.Repositories.List(ctx, tt.projectKey, nil)
			if err != nil {
				t.Fatalf("Repositores.List returned error: %v", err)
			}

			if diff := cmp.Diff(tt.repos, list.Repositories); diff != "" {
				t.Fatalf("Repositores.List returned diff (want -> got):\n%s", diff)
			}

		})
	}
}

func TestCreateRepository(t *testing.T) {
	tests := []struct {
		name       string
		repository Repository
	}{
		{
			name: "repository 1",
			repository: Repository{
				Name:     "repo1",
				ScmID:    "git",
				Forkable: true,
				Public:   true,
				Project: Project{
					Key: "prj1",
				},
			},
		},
		{
			name: "repository 2",
			repository: Repository{
				Name:     "repo2",
				ScmID:    "git",
				Forkable: true,
				Public:   true,
				Project: Project{
					Key: "prj2",
				},
			},
		},
		{
			name: "nil repo",
			repository: Repository{
				Project: Project{
					Key: "prj3",
				},
			},
		},
	}

	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// /rest/api/1.0/projects/{projectKey}/repos".
			path := fmt.Sprintf("%s/%s/%s/%s", stashURIprefix, projectsURI, tt.repository.Project.Key, RepositoriesURI)
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					req := &Repository{}
					json.NewDecoder(r.Body).Decode(req)
					if (req.ScmID == tt.repository.ScmID) && (req.Name == tt.repository.Name) {
						w.WriteHeader(http.StatusOK)
						r := &Repository{
							Slug:          tt.repository.Name,
							Name:          tt.repository.Name,
							ID:            1,
							State:         "AVAILABLE",
							StatusMessage: "Available",
							Public:        true,
							Project: Project{
								Key: tt.repository.Project.Key,
							},
						}
						json.NewEncoder(w).Encode(r)
						return
					}
				}

				http.Error(w, "The repository was not created due to a validation error", http.StatusBadRequest)

				return

			})

			ctx := context.Background()
			r, err := client.Repositories.Create(ctx, &tt.repository)
			if err != nil {
				if !strings.Contains(err.Error(), "validation error") {
					t.Fatalf("Repositories.Create returned error: %v", err)
				}
				return
			}

			if (r.Name != tt.repository.Name) || (r.Project.Key != tt.repository.Project.Key) {
				t.Errorf("Repositories.Create returned:\n%v, want:\n%v", r, tt.repository)
			}
		})
	}
}

func TestUpdateRepository(t *testing.T) {
	tests := []struct {
		name       string
		repository Repository
	}{
		{
			name: "update project",
			repository: Repository{
				Name: "repo1",
				Project: Project{
					Key: "prj2",
				},
			},
		},
		{
			name: "update name",
			repository: Repository{
				Name: "repo2",
				Project: Project{
					Key: "prj2",
				},
			},
		},
	}
	projectKey, repositorySlug := "prj1", "repo1"

	mux, client := setup(t)

	// /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}".
	path := fmt.Sprintf("%s/%s/%s/%s/%s", stashURIprefix, projectsURI, projectKey, RepositoriesURI, repositorySlug)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			req := &Repository{}
			json.NewDecoder(r.Body).Decode(req)
			r := &Repository{
				Slug:          repositorySlug,
				Name:          repositorySlug,
				ID:            1,
				State:         "AVAILABLE",
				StatusMessage: "Available",
				ScmID:         "git",
				Forkable:      true,
				Public:        true,
				Project: Project{
					Key: projectKey,
				},
			}
			if req.Name != "" {
				r.Name = req.Name
			}

			if req.Project.Key != "" {
				r.Project.Key = req.Project.Key
			}

			w.WriteHeader(http.StatusOK)

			json.NewEncoder(w).Encode(r)
			return
		}

		http.Error(w, "The repository was not updated due to a validation error", http.StatusBadRequest)

		return

	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r, err := client.Repositories.Update(ctx, projectKey, repositorySlug, &tt.repository)
			if err != nil {
				t.Fatalf("Repositories.Update returned error: %v", err)
			}

			if (r.Name != tt.repository.Name) || (r.Project.Key != tt.repository.Project.Key) {
				t.Errorf("Repositories.Update returned %v, want %v", r, tt.repository)
			}
		})
	}
}

func TestDeleteRepository(t *testing.T) {
	tests := []struct {
		name           string
		projectKey     string
		repositorySlug string
	}{
		{
			name:           "test repository does not exist",
			projectKey:     "prj",
			repositorySlug: "default",
		},
		{
			name:           "test a repository",
			projectKey:     "prj",
			repositorySlug: "repo1",
		},
	}

	validNames := []string{"repo1"}

	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// /rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}".
			path := fmt.Sprintf("%s/%s/%s/%s/%s", stashURIprefix, projectsURI, tt.projectKey, RepositoriesURI, tt.repositorySlug)
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				for _, substr := range validNames {
					if (r.Method == http.MethodDelete) && (strings.Contains(r.URL.String(), substr)) {
						w.WriteHeader(http.StatusAccepted)
						w.Write([]byte("202 - OK!"))
						return
					}
				}
				w.WriteHeader(http.StatusNoContent)
				w.Write([]byte("204 - No repository matching the supplied projectKey and repositorySlug was found"))

				return

			})
			ctx := context.Background()
			err := client.Repositories.Delete(ctx, tt.projectKey, tt.repositorySlug)
			if err != nil {
				t.Errorf("Repositories.Delete returned error: %v", err)
			}
		})
	}
}

func TestGetRepositoryGroupPermission(t *testing.T) {
	type group struct {
		Name string "json:\"name,omitempty\""
	}

	tests := []struct {
		name  string
		group *RepositoryGroupPermission
	}{
		{
			name: "test Reader permission",
			group: &RepositoryGroupPermission{
				Group: group{
					Name: "Reader",
				},
				Permission: "REPO_READ",
			},
		},
		{
			name: "test Writer permission",
			group: &RepositoryGroupPermission{
				Group: group{
					Name: "Writer",
				},
				Permission: "REPO_WRITE",
			},
		},
		{
			name: "test Admin permission",
			group: &RepositoryGroupPermission{
				Group: group{
					Name: "Admin",
				},
				Permission: "REPO_ADMIN",
			},
		},
	}

	mux, client := setup(t)

	///rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/permissions/groups?filter
	path := fmt.Sprintf("%s/%s/testProject/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, groupPermisionsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "testProject") && strings.Contains(r.RequestURI, "repo1") {
			g := struct {
				Groups []*RepositoryGroupPermission `json:"values"`
			}{
				[]*RepositoryGroupPermission{
					{},
				},
			}
			switch r.URL.Query().Get("filter") {
			case "Reader":
				g.Groups[0].Group.Name = "Reader"
				g.Groups[0].Permission = "REPO_READ"
			case "Writer":
				g.Groups[0].Group.Name = "Writer"
				g.Groups[0].Permission = "REPO_WRITE"
			case "Admin":
				g.Groups[0].Group.Name = "Admin"
				g.Groups[0].Permission = "REPO_ADMIN"
			default:
				http.Error(w, "The repository group was not found", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(g)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			g, err := client.Repositories.GetRepositoryGroupPermission(ctx, "testProject", "repo1", tt.group.Group.Name)
			if err != nil {
				t.Fatalf("Repositories.GetRepositoryGroupPermission returned error: %v", err)
			}

			if diff := cmp.Diff(tt.group, g); diff != "" {
				t.Errorf("Repositories.GetRepositoryGroupPermission returned (want -> got): %s", diff)
			}
		})
	}
}

func TestListRepositoryGroupsPermission(t *testing.T) {
	type group struct {
		Name string "json:\"name,omitempty\""
	}
	tests := []*RepositoryGroupPermission{
		{
			Group: group{
				Name: "Reader",
			},
			Permission: "REPO_READ",
		},
		{
			Group: group{
				Name: "Writer",
			},
			Permission: "REPO_WRITE",
		},
		{
			Group: group{
				Name: "Admin",
			},
			Permission: "REPO_ADMIN",
		},
	}

	mux, client := setup(t)

	///rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/permissions/groups
	path := fmt.Sprintf("%s/%s/testProject/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, groupPermisionsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "testProject") && strings.Contains(r.RequestURI, "repo1") {
			w.WriteHeader(http.StatusOK)
			g := struct {
				Groups []*RepositoryGroupPermission `json:"values"`
			}{
				[]*RepositoryGroupPermission{
					{
						Group:      tests[0].Group,
						Permission: tests[0].Permission,
					},
					{
						Group:      tests[1].Group,
						Permission: tests[1].Permission,
					},
					{
						Group:      tests[2].Group,
						Permission: tests[2].Permission,
					},
				},
			}

			json.NewEncoder(w).Encode(g)
			return
		}
	})
	ctx := context.Background()
	list, err := client.Repositories.ListRepositoryGroupsPermission(ctx, "testProject", "repo1", nil)
	if err != nil {
		t.Fatalf("Repositories.ListRepositoryGroupsPermission returned error: %v", err)
	}

	if diff := cmp.Diff(tests, list.Groups); diff != "" {
		t.Errorf("Repositories.ListRepositoryGroupsPermission returned (want -> got): %s", diff)
	}

}

func TestUpdateRepositoryGroupsPermission(t *testing.T) {
	type group struct {
		Name string "json:\"name,omitempty\""
	}

	tests := []struct {
		name  string
		group RepositoryGroupPermission
	}{
		{
			name: "test promote to Writer",
			group: RepositoryGroupPermission{
				Group: group{
					Name: "Reader",
				},
				Permission: "REPO_WRITE",
			},
		},
		{
			name: "test promote to Admin",
			group: RepositoryGroupPermission{
				Group: group{
					Name: "Reader",
				},
				Permission: "REPO_ADMIN",
			},
		},
		{
			name: "test promote to BDFL",
			group: RepositoryGroupPermission{
				Group: group{
					Name: "Reader",
				},
				Permission: "REPO_BDFL",
			},
		},
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/testProject/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, groupPermisionsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			req := &RepositoryGroupPermission{}
			json.NewDecoder(r.Body).Decode(req)
			switch req.Permission {
			case "Writer":
			case "Admin":
			default:
				http.Error(w, "The request was malformed or the specified permission does not exist", http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("204 - The requested permission was granted"))
			return
		}

		http.Error(w, "The permission was not updated due to a validation error", http.StatusBadRequest)

		return

	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := client.Repositories.UpdateRepositoryGroupPermission(ctx, "testProject", "repo1", &tt.group)
			if err != nil {
				if !strings.Contains(err.Error(), "The request was malformed") {
					t.Fatalf("UpdateRepositoryGroupPermission returned error: %v", err)
				}
				return
			}
		})
	}
}

func TestListRepositoryUsersPermission(t *testing.T) {
	tests := []*RepositoryUserPermission{
		{
			User: User{
				Name: "Jane Citizen",
				Slug: "jcitizen",
			},
			Permission: "REPO_READ",
		},
		{
			User: User{
				Name: "James Cook",
				Slug: "jcook",
			},
			Permission: "REPO_WRITE",
		},
		{
			User: User{
				Name: "Jack Sparrow",
				Slug: "jsparrow",
			},
			Permission: "REPO_ADMIN",
		},
	}

	mux, client := setup(t)

	///rest/api/1.0/projects/{projectKey}/repos/{repositorySlug}/permissions/groups
	path := fmt.Sprintf("%s/%s/testProject/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, userPermisionsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "testProject") && strings.Contains(r.RequestURI, "repo1") {
			w.WriteHeader(http.StatusOK)
			g := struct {
				Groups []*RepositoryUserPermission `json:"values"`
			}{
				[]*RepositoryUserPermission{
					{
						User:       tests[0].User,
						Permission: tests[0].Permission,
					},
					{
						User:       tests[1].User,
						Permission: tests[1].Permission,
					},
					{
						User:       tests[2].User,
						Permission: tests[2].Permission,
					},
				},
			}

			json.NewEncoder(w).Encode(g)
			return
		}
	})
	ctx := context.Background()
	list, err := client.Repositories.ListRepositoryUsersPermission(ctx, "testProject", "repo1", nil)
	if err != nil {
		t.Fatalf("Repositories.ListRepositoryUsersPermission returned error: %v", err)
	}

	if diff := cmp.Diff(tests, list.Users); diff != "" {
		t.Errorf("Repositories.ListRepositoryUsersPermission returned (want -> got): %s", diff)
	}

}
