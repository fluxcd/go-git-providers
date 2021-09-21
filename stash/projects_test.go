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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetProject(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		output      string
	}{
		{
			name:        "test project does not exist",
			projectName: "admin",
		},
		{
			name:        "test a project",
			projectName: "project 1",
		},
	}

	validNames := []string{"project1"}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s", stashURIprefix, projectsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		for _, substr := range validNames {
			if strings.Contains(r.Form.Get("name"), substr) {
				w.WriteHeader(http.StatusOK)
				u := &Project{
					Name: substr,
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		}

		http.Error(w, "The specified project does not exist", http.StatusNotFound)

		return

	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			group, err := client.Projects.Get(ctx, tt.projectName)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("Projects.GetProject returned error: %v", err)
				}
				return
			}

			if group.Name != tt.projectName {
				t.Errorf("Projects.GetProject returned group %s, want %s", group.Name, tt.projectName)
			}

		})
	}
}

func TestListProjects(t *testing.T) {
	pNames := []*Project{
		{Name: "project1"}, {Name: "demo"}, {Name: "infra"}, {Name: "app"}}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s", stashURIprefix, projectsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		u := struct {
			Projects []*Project `json:"values"`
		}{[]*Project{
			pNames[0],
			pNames[1],
			pNames[2],
			pNames[3],
		}}
		json.NewEncoder(w).Encode(u)
		return

	})
	ctx := context.Background()
	list, err := client.Projects.List(ctx, nil)
	if err != nil {
		t.Fatalf("Projects.ProjectList returned error: %v", err)
	}

	if diff := cmp.Diff(pNames, list.Projects); diff != "" {
		t.Errorf("Projects.ProjectList returned diff (want -> got):\n%s", diff)
	}

}

func TestListProjectGroupsPermission(t *testing.T) {

	type group struct {
		Name string "json:\"name,omitempty\""
	}
	tests := []*ProjectGroupPermission{
		{
			Group: group{
				Name: "Reader",
			},
			Permission: "PROJECT_READ",
		},
		{
			Group: group{
				Name: "Writer",
			},
			Permission: "PROJECT_WRITE",
		},
		{
			Group: group{
				Name: "Admin",
			},
			Permission: "PROJECT_ADMIN",
		},
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/projects/testProject/%s", stashURIprefix, groupPermisionsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "testProject") {
			w.WriteHeader(http.StatusOK)
			g := struct {
				Groups []*ProjectGroupPermission `json:"values"`
			}{
				[]*ProjectGroupPermission{
					tests[0],
					tests[1],
					tests[2],
				},
			}

			json.NewEncoder(w).Encode(g)
			return
		}
	})
	ctx := context.Background()
	list, err := client.Projects.ListProjectGroupsPermission(ctx, "testProject", nil)
	if err != nil {
		t.Fatalf("Projects.ListProjectGroupsPermission returned error: %v", err)
	}

	if diff := cmp.Diff(tests, list.Groups); diff != "" {
		t.Errorf("Projects.ListProjectGroupsPermission returned diff (want -> got):\n%s", diff)
	}
}

func TestListProjectUsersPermission(t *testing.T) {
	tests := []*ProjectUserPermission{
		{
			User: User{
				Slug: "Reader",
			},
			Permission: "PROJECT_READ",
		},
		{
			User: User{
				Slug: "Writer",
			},
			Permission: "PROJECT_WRITE",
		},
		{
			User: User{
				Slug: "Admin",
			},
			Permission: "PROJECT_ADMIN",
		},
	}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/projects/testProject/%s", stashURIprefix, userPermisionsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "testProject") {
			w.WriteHeader(http.StatusOK)
			g := struct {
				Groups []*ProjectUserPermission `json:"values"`
			}{
				[]*ProjectUserPermission{
					tests[0],
					tests[1],
					tests[2],
				},
			}

			json.NewEncoder(w).Encode(g)
			return
		}
	})
	ctx := context.Background()
	list, err := client.Projects.ListProjectUsersPermission(ctx, "testProject", nil)
	if err != nil {
		t.Fatalf("Projects.ListProjectUsersPermission returned error: %v", err)
	}

	if diff := cmp.Diff(tests, list.Users); diff != "" {
		t.Errorf("Projects.ListProjectUsersPermission returned diff (want -> got):\n%s", diff)
	}

}
