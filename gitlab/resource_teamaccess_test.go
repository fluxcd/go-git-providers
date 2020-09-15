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

func Test_getPermissionFromMap(t *testing.T) {
	tests := []struct {
		name        string
		permissions map[int]gitprovider.RepositoryPermission
		want        *gitprovider.RepositoryPermission
	}{
		{
			name: "pull",
			permissions: map[int]gitprovider.RepositoryPermission{
				10: gitprovider.RepositoryPermissionPull,
				20: gitprovider.RepositoryPermissionTriage,
				30: gitprovider.RepositoryPermissionPush,
				40: gitprovider.RepositoryPermissionMaintain,
				50: gitprovider.RepositoryPermissionAdmin,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
		},
		{
			name: "push",
			permissions: map[int]gitprovider.RepositoryPermission{
				10: gitprovider.RepositoryPermissionPull,
				20: gitprovider.RepositoryPermissionTriage,
				30: gitprovider.RepositoryPermissionPush,
				40: gitprovider.RepositoryPermissionMaintain,
				50: gitprovider.RepositoryPermissionAdmin,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
		},
		{
			name: "admin",
			permissions: map[int]gitprovider.RepositoryPermission{
				10: gitprovider.RepositoryPermissionPull,
				20: gitprovider.RepositoryPermissionTriage,
				30: gitprovider.RepositoryPermissionPush,
				40: gitprovider.RepositoryPermissionMaintain,
				50: gitprovider.RepositoryPermissionAdmin,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
		},
		{
			name: "none",
			permissions: map[int]gitprovider.RepositoryPermission{
				10: gitprovider.RepositoryPermissionPull,
				20: gitprovider.RepositoryPermissionTriage,
				30: gitprovider.RepositoryPermissionPush,
				40: gitprovider.RepositoryPermissionMaintain,
				50: gitprovider.RepositoryPermissionAdmin,
			},
			want: nil,
		},
		{
			name: "false data",
			permissions: map[int]gitprovider.RepositoryPermission{
				10: gitprovider.RepositoryPermissionPull,
				20: gitprovider.RepositoryPermissionTriage,
				30: gitprovider.RepositoryPermissionPush,
				40: gitprovider.RepositoryPermissionMaintain,
				50: gitprovider.RepositoryPermissionAdmin,
			},
			want: nil,
		},
		{
			name: "not all specified",
			permissions: map[int]gitprovider.RepositoryPermission{
				10: gitprovider.RepositoryPermissionPull,
				20: gitprovider.RepositoryPermissionTriage,
				30: gitprovider.RepositoryPermissionPush,
				40: gitprovider.RepositoryPermissionMaintain,
				50: gitprovider.RepositoryPermissionAdmin,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPermission := getPermissionFromMap(tt.permissions)
			if !reflect.DeepEqual(gotPermission, tt.want) {
				t.Errorf("getPermissionFromMap() = %v, want %v", gotPermission, tt.want)
			}
		})
	}
}
