//go:build e2e

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

package gitlab

import (
	"reflect"
	"testing"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

func Test_getGitProviderPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission int
		want       *gitprovider.RepositoryPermission
	}{
		{
			name:       "pull",
			permission: 10,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
		},
		{
			name:       "push",
			permission: 30,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
		},
		{
			name:       "admin",
			permission: 50,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
		},
		{
			name:       "false data",
			permission: -1,
			want:       nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPermission, _ := getGitProviderPermission(tt.permission)
			if !reflect.DeepEqual(gotPermission, tt.want) {
				t.Errorf("getPermissionFromMap() = %v, want %v", gotPermission, tt.want)
			}
		})
	}
}

func Test_getGitlabPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission *gitprovider.RepositoryPermission
		want       int
	}{
		{
			name:       "pull",
			permission: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
			want:       10,
		},
		{
			name:       "push",
			permission: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
			want:       30,
		},
		{
			name:       "admin",
			permission: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
			want:       50,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPermission, _ := getGitlabPermission(*tt.permission)
			if !reflect.DeepEqual(gotPermission, tt.want) {
				t.Errorf("getPermissionFromMap() = %v, want %v", gotPermission, tt.want)
			}
		})
	}
}
