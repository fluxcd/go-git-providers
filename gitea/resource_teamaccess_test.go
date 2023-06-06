/*
Copyright 2023 The Flux CD contributors.

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

package gitea

import (
	"reflect"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/fluxcd/go-git-providers/gitprovider"
)

func Test_getGitProviderPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission gitea.AccessMode
		want       *gitprovider.RepositoryPermission
	}{
		{
			name:       "pull",
			permission: gitea.AccessModeRead,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
		},
		{
			name:       "push",
			permission: gitea.AccessModeWrite,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
		},
		{
			name:       "admin",
			permission: gitea.AccessModeAdmin,
			want:       gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPermission := getProviderPermission(tt.permission)
			if !reflect.DeepEqual(gotPermission, tt.want) {
				t.Errorf("getPermissionFromMap() = %v, want %v", gotPermission, tt.want)
			}
		})
	}
}
