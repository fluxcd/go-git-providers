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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetCommit(t *testing.T) {
	tests := []struct {
		name     string
		commitID string
	}{
		{
			name:     "test a commit",
			commitID: "abcdef0123abcdef4567abcdef8987abcdef6543",
		},
		{
			name:     "test commit does not exist",
			commitID: "*°0#13jbkjfbvsqbùbjùrdfbgzeo'àtu)éuçt&-y",
		},
	}

	validCommitID := []string{"abcdef0123abcdef4567abcdef8987abcdef6543"}

	mux, client := setup(t)

	fmt.Println("commit")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s/%s", stashURIprefix, projectsURI, RepositoriesURI, commitsURI, tt.commitID)
			mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
				for _, substr := range validCommitID {
					if path.Base(r.URL.String()) == substr {
						w.WriteHeader(http.StatusOK)
						c := &CommitObject{
							ID:        substr,
							DisplayID: substr[:10],
						}

						json.NewEncoder(w).Encode(c)
						return
					}
				}

				http.Error(w, "The specified commit does not exist", http.StatusNotFound)

				return

			})

			ctx := context.Background()
			c, err := client.Commits.Get(ctx, "prj1", "repo1", tt.commitID)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("Commits.Get returned error: %v", err)
				}
				return
			}

			if c.ID != tt.commitID {
				t.Fatalf("Commits.Get returned commit %s, want %s", c.ID, tt.commitID)
			}

		})
	}
}

func TestListCommits(t *testing.T) {
	cIDs := []*CommitObject{
		{ID: "abcdef0123abcdef4567abcdef8987abcdef6543"},
		{ID: "aerfdef09893abcdef4567abcdef898abcdef652"},
		{ID: "abcdef3456abcdef4567abcdef8987abcdef6657"},
		{ID: "abcdef9876abcdef4567abcdef8987abcdef4357"}}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s/prj1/%s/repo1/%s", stashURIprefix, projectsURI, RepositoriesURI, commitsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b := struct {
			Commits []*CommitObject `json:"values"`
		}{[]*CommitObject{
			cIDs[0],
			cIDs[1],
			cIDs[2],
			cIDs[3],
		}}
		json.NewEncoder(w).Encode(b)
		return

	})
	ctx := context.Background()
	list, err := client.Commits.List(ctx, "prj1", "repo1", nil)
	if err != nil {
		t.Fatalf("Commits.List returned error: %v", err)
	}

	if diff := cmp.Diff(cIDs, list.Commits); diff != "" {
		t.Errorf("Commits.List returned diff (want -> got):\n%s", diff)
	}

}
