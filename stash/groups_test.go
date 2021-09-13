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

func TestGetGroup(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		output    string
	}{
		{
			name:      "test group does not exist",
			groupName: "admin",
		},
		{
			name:      "test a group",
			groupName: "dev-1",
		},
	}

	validNames := []string{"dev-1"}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s", stashURIprefix, groupsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		for _, substr := range validNames {
			if strings.Contains(r.Form.Get("filter"), substr) {
				w.WriteHeader(http.StatusOK)
				u := &Group{
					Name: substr,
				}
				json.NewEncoder(w).Encode(u)
				return
			}
		}

		http.Error(w, "The specified group does not exist", http.StatusNotFound)

		return

	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			group, err := client.Groups.Get(ctx, tt.groupName)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("Groups.GetGroup returned error: %v", err)
				}
				return
			}

			if group.Name != tt.groupName {
				t.Errorf("Groups.GetGroup returned group %s, want %s", group.Name, tt.groupName)
			}

		})
	}
}

func TestListGroup(t *testing.T) {
	gNames := []*Group{
		{Name: "avengers"}, {Name: "x-men"}, {Name: "ff4"}, {Name: "x-team"}}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s", stashURIprefix, groupsURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		u := struct {
			Groups []*Group `json:"values"`
		}{[]*Group{
			gNames[0],
			gNames[1],
			gNames[2],
			gNames[3],
		}}
		json.NewEncoder(w).Encode(u)
		return

	})
	ctx := context.Background()
	list, err := client.Groups.List(ctx, nil)
	if err != nil {
		t.Fatalf("Groups.GetGroupList returned error: %v", err)
	}

	if diff := cmp.Diff(gNames, list.Groups); diff != "" {
		t.Errorf("Groups.List returned diff (want -> got):\n%s", diff)
	}

}

func TestListGroupMembers(t *testing.T) {
	wants := []*User{
		{Name: "John Citizen", Slug: "jcitizen"},
		{Name: "Tony Stark", Slug: "tstark"},
		{Name: "Reed Richards", Slug: "rrichards"},
		{Name: "Riri Williams", Slug: "rwilliams"}}

	mux, client := setup(t)

	path := fmt.Sprintf("%s/%s", stashURIprefix, groupMembersURI)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.RequestURI, "testGroup") {
			w.WriteHeader(http.StatusOK)
			u := struct {
				Users []*User `json:"values"`
			}{[]*User{
				wants[0],
				wants[1],
				wants[2],
				wants[3],
			}}
			json.NewEncoder(w).Encode(u)
			return
		}

	})
	ctx := context.Background()
	list, err := client.Groups.ListGroupMembers(ctx, "testGroup", nil)
	if err != nil {
		t.Fatalf("Groups.ListGroupMembers returned error: %v", err)
	}

	if diff := cmp.Diff(wants, list.Users); diff != "" {
		t.Errorf("Groups.ListGroupMembers returned diff (want -> got):\n%s", diff)
	}
}
