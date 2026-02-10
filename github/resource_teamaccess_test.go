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

package github

import (
	"reflect"
	"testing"

	"github.com/google/go-github/v82/github"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func Test_getPermissionFromMap(t *testing.T) {
	trueValue := true
	falseValue := false

	tests := []struct {
		name        string
		permissions *github.RepositoryPermissions
		want        *gitprovider.RepositoryPermission
	}{
		{
			name: "pull",
			permissions: &github.RepositoryPermissions{
				Pull:     &trueValue,
				Triage:   &falseValue,
				Push:     &falseValue,
				Maintain: &falseValue,
				Admin:    &falseValue,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
		},
		{
			name: "push",
			permissions: &github.RepositoryPermissions{
				Triage:   &falseValue,
				Push:     &trueValue,
				Maintain: &falseValue,
				Pull:     &trueValue,
				Admin:    &falseValue,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
		},
		{
			name: "admin",
			permissions: &github.RepositoryPermissions{
				Admin:    &trueValue,
				Pull:     &trueValue,
				Triage:   &trueValue,
				Maintain: &trueValue,
				Push:     &trueValue,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
		},
		{
			name: "none",
			permissions: &github.RepositoryPermissions{
				Admin:    &falseValue,
				Pull:     &falseValue,
				Push:     &falseValue,
				Maintain: &falseValue,
				Triage:   &falseValue,
			},
			want: nil,
		},
		{
			name: "false data",
			permissions: &github.RepositoryPermissions{
				Pull:     &falseValue,
				Triage:   &falseValue,
				Push:     &falseValue,
				Maintain: &falseValue,
				Admin:    &falseValue,
			},
			want: nil,
		},
		{
			name: "not all specifed",
			permissions: &github.RepositoryPermissions{
				Pull:     &falseValue,
				Triage:   &falseValue,
				Push:     &falseValue,
				Maintain: &falseValue,
				Admin:    &falseValue,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPermission := getPermissionFromStruct(tt.permissions)
			if !reflect.DeepEqual(gotPermission, tt.want) {
				t.Errorf("getPermissionFromMap() = %v, want %v", gotPermission, tt.want)
			}
		})
	}
}
