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
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestGetPR(t *testing.T) {
	tests := []struct {
		name string
		prID int
	}{
		{
			name: "test a pull request",
			prID: 101,
		},
		{
			name: "test pull request does not exist",
			prID: -1,
		},
	}

	validPRID := []string{"101"}

	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s/%s", stashURIprefix, projectsURI, RepositoriesURI, pullRequestsURI, strconv.Itoa(tt.prID))
			mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				for _, substr := range validPRID {
					if path.Base(r.URL.String()) == substr {
						w.WriteHeader(http.StatusOK)
						c := &PullRequest{
							IDVersion: IDVersion{
								ID: 101,
							},
							Author: Participant{
								User: User{
									Name: "test",
								},
								Role: "AUTHOR",
							},
							FromRef: Ref{
								ID: "refs/heads/feature-ABC-123",
							},
							ToRef: Ref{
								ID: "refs/heads/main",
							},
							Properties: Properties{
								MergeResult: MergeResult{
									Current: true,
									Outcome: "SUCCESS",
								},
							},
						}

						json.NewEncoder(w).Encode(c)
						return
					}
				}

				http.Error(w, "The specified pr does not exist", http.StatusNotFound)

				return

			})

			ctx := context.Background()
			c, err := client.PullRequests.Get(ctx, "prj1", "repo1", tt.prID)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("PullRequest.Get returned error: %v", err)
				}
				return
			}

			if c.ID != tt.prID {
				t.Fatalf("PullRequest.Get returned:\n%d, want:\n%d", c.ID, tt.prID)
			}

		})
	}
}

func TestListPRs(t *testing.T) {
	prIDs := []*PullRequest{
		{IDVersion: IDVersion{ID: 101}},
		{IDVersion: IDVersion{ID: 102}},
		{IDVersion: IDVersion{ID: 103}},
		{IDVersion: IDVersion{ID: 104}},
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, pullRequestsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b := struct {
			PRs []*PullRequest `json:"values"`
		}{[]*PullRequest{
			prIDs[0],
			prIDs[1],
			prIDs[2],
			prIDs[3],
		}}
		json.NewEncoder(w).Encode(b)
		return

	})
	ctx := context.Background()
	list, err := client.PullRequests.List(ctx, "prj1", "repo1", nil)
	if err != nil {
		t.Fatalf("PullRequests.List returned error: %v", err)
	}

	if diff := cmp.Diff(prIDs, list.PullRequests); diff != "" {
		t.Errorf("PullRequests.List returned diff (want -> got):\n%s", diff)
	}

}

func TestCreatePR(t *testing.T) {
	tests := []struct {
		name string
		pr   CreatePullRequest
	}{
		{
			name: "pr 1",
			pr: CreatePullRequest{
				Title:       "PR service",
				Description: "A service that manages prs.",
				State:       "OPEN",
				Open:        true,
				Closed:      false,
				FromRef: Ref{
					ID: "refs/heads/feature-pr",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
				ToRef: Ref{
					ID: "refs/heads/main",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
				Locked: false,
				Reviewers: []User{
					{
						Name: "charlie",
					},
				},
			},
		},
		{
			name: "pr 2",
			pr: CreatePullRequest{
				Title:       "PR service",
				Description: "A service that manages prs.",
				State:       "OPEN",
				Open:        true,
				Closed:      false,
				FromRef: Ref{
					ID: "refs/heads/feature-pr",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
				ToRef: Ref{
					ID: "refs/heads/main",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
				Locked: false,
				Reviewers: []User{
					{
						Name: "charlie",
					},
				},
			},
		},
		{
			name: "invalid pr",
			pr: CreatePullRequest{
				Title:       "Invalid PR",
				Description: "This PR is invalid because the ToRef and FromRef are the same branches.",
				State:       "OPEN",
				Open:        true,
				Closed:      false,
				FromRef: Ref{
					ID: "refs/heads/main",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
				ToRef: Ref{
					ID: "refs/heads/main",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
				Locked: false,
				Reviewers: []User{
					{
						Name: "charlie",
					},
				},
			},
		},
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj/%s/my-repo/%s", stashURIprefix, projectsURI, RepositoriesURI, pullRequestsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			req := &CreatePullRequest{}
			json.NewDecoder(r.Body).Decode(req)
			if req.FromRef.ID != req.ToRef.ID {
				w.WriteHeader(http.StatusOK)
				r := &PullRequest{
					IDVersion: IDVersion{
						ID:      1,
						Version: 1,
					},
					CreatedDate: time.Now().Unix(),
					UpdatedDate: time.Now().Unix(),
					Title:       req.Title,
					Description: req.Description,
					State:       req.State,
					Open:        req.Open,
					Closed:      req.Closed,
					FromRef:     req.FromRef,
					ToRef:       req.ToRef,
					Locked:      req.Locked,
					Author: Participant{
						User: User{
							Name: "Rob",
						},
					},
					Reviewers: []Participant{
						{
							User:     req.Reviewers[0],
							Role:     "REVIEWER",
							Approved: false,
							Status:   "UNAPPROVED",
						},
					},
				}
				json.NewEncoder(w).Encode(r)
				return
			}
		}

		http.Error(w, "The pull request was not created due to same specified branches.", http.StatusConflict)

		return

	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			p, err := client.PullRequests.Create(ctx, "prj", "my-repo", &tt.pr)
			if err != nil {
				if !strings.Contains(err.Error(), "409 Conflict") {
					t.Fatalf("PullRequest.Create returned error: %v", err)
				}
				return
			}

			if (p.Title != tt.pr.Title) || (p.FromRef.ID != tt.pr.FromRef.ID) || (p.ToRef.ID != tt.pr.ToRef.ID) {
				t.Errorf("PullRequest.Create returned:\n%v, want:\n%v", p, tt.pr)
			}
		})
	}
}

func TestUpdatePR(t *testing.T) {
	tests := []struct {
		name string
		pr   PullRequest
	}{
		{
			name: "update description",
			pr: PullRequest{
				IDVersion: IDVersion{
					ID:      1,
					Version: 2,
				},
				Title:       "PR service",
				Description: "A service that manages prs. It supports get, list, create, update and deletes ops.",
				ToRef: Ref{
					ID: "refs/heads/main",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
			},
		},
		{
			name: "update destination branch",
			pr: PullRequest{
				IDVersion: IDVersion{
					ID:      1,
					Version: 3,
				},
				Title:       "PR service",
				Description: "A service that manages prs. It supports get, list, create, update and deletes ops.",
				ToRef: Ref{
					ID: "refs/heads/develop",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
			},
		},
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj/%s/my-repo/%s/%s", stashURIprefix, projectsURI, RepositoriesURI, pullRequestsURI, strconv.Itoa(tests[0].pr.ID))
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			req := &PullRequest{}
			json.NewDecoder(r.Body).Decode(req)
			r := &PullRequest{
				IDVersion: IDVersion{
					ID:      req.ID,
					Version: req.Version,
				},
				CreatedDate: time.Now().Unix(),
				UpdatedDate: time.Now().Unix(),
				Title:       req.Title,
				Description: req.Description,
				FromRef: Ref{
					ID: "refs/heads/feature-pr",
					Repository: Repository{
						Slug: "my-repo",
						Project: Project{
							Key: "prj",
						},
					},
				},
				ToRef: req.ToRef,
				Author: Participant{
					User: User{
						Name: "Rob",
					},
				},
				Reviewers: []Participant{
					{
						User: User{
							Name: "Charlie",
						},
						Role:     "REVIEWER",
						Approved: false,
						Status:   "UNAPPROVED",
					},
				},
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
			p, err := client.PullRequests.Update(ctx, "prj", "my-repo", &tt.pr)
			if err != nil {
				t.Fatalf("PullRequests.Update returned error: %v", err)
			}

			if (p.Title != tt.pr.Title) || (p.Description != tt.pr.Description) || (p.ToRef.ID != tt.pr.ToRef.ID) {
				t.Errorf("PullRequests.Update returned:\n%v, want:\n%v", p, tt.pr)
			}
		})
	}
}

func TestDeletePR(t *testing.T) {
	tests := []struct {
		name      string
		idVersion IDVersion
	}{
		{
			name: "test PR does not exist",
			idVersion: IDVersion{
				ID:      -1,
				Version: 2,
			},
		},
		{
			name: "test a PR",
			idVersion: IDVersion{
				ID:      1,
				Version: 1,
			},
		},
	}

	mux, client := setup(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s/prj/%s/my-repo/%s/%s", stashURIprefix, projectsURI, RepositoriesURI, pullRequestsURI, strconv.Itoa(tt.idVersion.ID))
			mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				id, err := strconv.ParseInt(path.Base(r.URL.String()), 10, 64)
				if err != nil {
					t.Fatalf("strconv.ParseInt returned error: %v", err)
				}
				if id >= 0 {
					w.WriteHeader(http.StatusNoContent)
					w.Write([]byte("204 - OK!"))
					return
				}

				http.Error(w, "The specified repository or pull request does not exist.", http.StatusNotFound)

				return

			})
			ctx := context.Background()
			err := client.PullRequests.Delete(ctx, "prj", "my-repo", tt.idVersion)
			if err != nil {
				if err != ErrNotFound {
					t.Errorf("PullRequests.Delete returned error: %v", err)
				}
			}
		})
	}
}
