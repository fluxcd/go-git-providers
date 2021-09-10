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

package gostash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setup sets up a test HTTP server along with a Client configured to talk to that test server.
// Tests should register handlers on mux which provide mock responses for the API method being tested.
func setup(t *testing.T) (*http.ServeMux, *httptest.Server, *Client) {
	mux := http.NewServeMux()
	// Start a local HTTP server
	server := httptest.NewServer(mux)
	// declare a Client
	berearHeader := &http.Header{
		"WWW-Authenticate": []string{"Bearer"},
	}
	client, err := NewClient(nil, server.URL, berearHeader, initLogger(t))
	if err != nil {
		server.Close()
		t.Errorf("unexpected error while declaring a client: %v", err)
	}
	return mux, server, client
}

// teardown closes the test HTTP server.
func teardown(server *httptest.Server) {
	server.Close()
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name   string
		slug   string
		output string
	}{
		{
			name: "test user does not exist",
			slug: "admin",
		},
		{
			name: "test a user",
			slug: "jcitizen",
		},
	}

	validSlugs := []string{"jcitizen"}

	mux, server, client := setup(t)
	defer teardown(server)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/users/%s", stashURIprefix, tt.slug)
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				for _, substr := range validSlugs {
					if strings.Contains(r.URL.Path, substr) {
						w.WriteHeader(http.StatusOK)
						u := &User{
							Slug: substr,
						}
						json.NewEncoder(w).Encode(u)
						return
					}
				}

				http.Error(w, "The specified user does not exist", http.StatusNotFound)

				return

			})

			ctx := context.Background()
			user, err := client.Users.GetUser(ctx, tt.slug)
			if err != nil {
				if err != ErrNotFound {
					t.Fatalf("Users.GetUser returned error: %v", err)
				}
			} else {
				if user.Slug != tt.slug {
					t.Errorf("Users.GetUser returned user %s, want %s", user.Slug, tt.slug)
				}
			}

		})
	}
}

func TestUserList(t *testing.T) {
	slugs := []string{"jcitizen", "tstark", "rrichards", "rwilliams"}

	mux, server, client := setup(t)
	defer teardown(server)

	path := fmt.Sprintf("%s/users", stashURIprefix)
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)
		u := struct {
			Users []User `json:"values"`
		}{[]User{
			{
				Name: "John Citizen",
				Slug: slugs[0],
			},
			{
				Name: "Tony Stark",
				Slug: slugs[1],
			},
			{
				Name: "Reed Richards",
				Slug: slugs[2],
			},
			{
				Name: "Riri Williams",
				Slug: slugs[3],
			},
		}}
		json.NewEncoder(w).Encode(u)
		return
	})

	ctx := context.Background()
	list, err := client.Users.List(ctx, nil)
	if err != nil {
		if err != ErrNotFound {
			t.Fatalf("Users.GetUser returned error: %v", err)
		}
	}

	if len(list.Users) != len(slugs) {
		t.Fatalf("Users.GetUser returned %d users, want %d", len(list.Users), len(slugs))
	}
}
