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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetBranch(t *testing.T) {
	tests := []struct {
		name     string
		branchID string
	}{
		{
			name:     "test branch does not exist",
			branchID: "features",
		},
		{
			name:     "test main branch",
			branchID: "refs/heads/main",
		},
	}

	validBranchID := []string{"refs/heads/main"}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, branchesURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		for _, substr := range validBranchID {
			if r.URL.Query().Get("filterText") == substr {
				w.WriteHeader(http.StatusOK)
				u := &Branch{
					ID:        "refs/heads/main",
					DisplayID: "main",
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		}

		http.Error(w, "The specified branch does not exist", http.StatusNotFound)

		return

	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b, err := client.Branches.Get(ctx, "prj1", "repo1", tt.branchID)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("Branches.Get returned error: %v", err)
				}
				return
			}

			if b.ID != tt.branchID {
				t.Errorf("Branches.Get returned branch:\n%s, want:\n%s", b.ID, tt.branchID)
			}

		})
	}
}

func TestListBranches(t *testing.T) {
	bIDs := []*Branch{
		{ID: "refs/heads/main"}, {ID: "refs/heads/release"}, {ID: "refs/heads/feature"}, {ID: "refs/heads/hotfix"}}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, branchesURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b := struct {
			Branches []*Branch `json:"values"`
		}{[]*Branch{
			bIDs[0],
			bIDs[1],
			bIDs[2],
			bIDs[3],
		}}
		json.NewEncoder(w).Encode(b)
		return

	})
	ctx := context.Background()
	list, err := client.Branches.List(ctx, "prj1", "repo1", nil)
	if err != nil {
		t.Fatalf("Branches.List returned error: %v", err)
	}

	if diff := cmp.Diff(bIDs, list.Branches); diff != "" {
		t.Errorf("Branches.List returned diff (want -> got):\n%s", diff)
	}

}

func TestDefaultBranch(t *testing.T) {
	d := struct {
		ID        string `json:"id"`
		DisplayID string `json:"displayId"`
	}{
		"refs/heads/main",
		"main",
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s/%s", stashURIprefix, projectsURI, RepositoriesURI, branchesURI, defaultBranchURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		u := &Branch{
			ID:        d.ID,
			DisplayID: d.DisplayID,
		}
		json.NewEncoder(w).Encode(u)

		return
	})

	ctx := context.Background()
	b, err := client.Branches.Default(ctx, "prj1", "repo1")
	if err != nil {
		if err != ErrNotFound {
			t.Fatalf("Branches.Default returned error: %v", err)
		}
		return
	}

	if b.ID != d.ID {
		t.Errorf("Branches.Default returned branch:\n%s, want:\n %s", b.ID, d.ID)
	}
}
