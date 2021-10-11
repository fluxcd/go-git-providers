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
	"reflect"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func Test_getGitProviderPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		want       *gitprovider.RepositoryPermission
	}{
		{
			name:       "pull",
			permission: stashPermissionRead,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
		},
		{
			name:       "push",
			permission: stashPermissionWrite,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
		},
		{
			name:       "admin",
			permission: stashPermissionAdmin,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
		},
		{
			name:       "false data",
			permission: "",
			want:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perMap := make(map[string]bool)
			perMap[tt.permission] = true
			gotPermission, _ := getGitProviderPermission(getStashPermissionFromMap(perMap))
			if gotPermission != nil && tt.want != nil && !reflect.DeepEqual(gotPermission, tt.want) {
				t.Errorf("getPermissionFromMap() = %v, want %v", *gotPermission, *tt.want)
			}
		})
	}
}

func Test_getStashPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission *gitprovider.RepositoryPermission
		want       string
	}{
		{
			name:       "pull",
			permission: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
			want:       "REPO_READ",
		},
		{
			name:       "push",
			permission: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
			want:       "REPO_WRITE",
		},
		{
			name:       "admin",
			permission: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
			want:       "REPO_ADMIN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPermission, _ := getStashPermission(*tt.permission)
			if !reflect.DeepEqual(gotPermission, tt.want) {
				t.Errorf("getPermissionFromMap() = %v, want %v", gotPermission, tt.want)
			}
		})
	}
}
