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

package gitprovider

import (
	"reflect"
	"testing"
)

func TestDefaulting(t *testing.T) {
	tests := []struct {
		name       string
		structName string
		object     CreatableInfo
		expected   CreatableInfo
	}{
		{
			name:       "DeployKey: empty",
			structName: "DeployKey",
			object:     &DeployKeyInfo{},
			expected: &DeployKeyInfo{
				ReadOnly: BoolVar(true),
			},
		},
		{
			name:       "DeployKey: don't set if non-nil (default)",
			structName: "DeployKey",
			object: &DeployKeyInfo{
				ReadOnly: BoolVar(true),
			},
			expected: &DeployKeyInfo{
				ReadOnly: BoolVar(true),
			},
		},
		{
			name:       "DeployKey: don't set if non-nil (non-default)",
			structName: "DeployKey",
			object: &DeployKeyInfo{
				ReadOnly: BoolVar(false),
			},
			expected: &DeployKeyInfo{
				ReadOnly: BoolVar(false),
			},
		},
		{
			name:       "Repository: empty",
			structName: "Repository",
			object:     &RepositoryInfo{},
			expected: &RepositoryInfo{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityPrivate),
				DefaultBranch: StringVar("master"),
			},
		},
		{
			name:       "Repository: don't set if non-nil (default)",
			structName: "Repository",
			object: &RepositoryInfo{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityPrivate),
				DefaultBranch: StringVar("master"),
			},
			expected: &RepositoryInfo{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityPrivate),
				DefaultBranch: StringVar("master"),
			},
		},
		{
			name:       "Repository: don't set if non-nil (non-default)",
			structName: "Repository",
			object: &RepositoryInfo{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityInternal),
				DefaultBranch: StringVar("main"),
			},
			expected: &RepositoryInfo{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityInternal),
				DefaultBranch: StringVar("main"),
			},
		},
		{
			name:       "TeamAccess: empty",
			structName: "TeamAccess",
			object:     &TeamAccessInfo{},
			expected: &TeamAccessInfo{
				Permission: RepositoryPermissionVar(RepositoryPermissionPull),
			},
		},
		{
			name:       "TeamAccess: don't set if non-nil (default)",
			structName: "Repository",
			object: &TeamAccessInfo{
				Permission: RepositoryPermissionVar(RepositoryPermissionPull),
			},
			expected: &TeamAccessInfo{
				Permission: RepositoryPermissionVar(RepositoryPermissionPull),
			},
		},
		{
			name:       "TeamAccess: don't set if non-nil (non-default)",
			structName: "TeamAccess",
			object: &TeamAccessInfo{
				Permission: RepositoryPermissionVar(RepositoryPermissionPush),
			},
			expected: &TeamAccessInfo{
				Permission: RepositoryPermissionVar(RepositoryPermissionPush),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.object.Default()

			if !reflect.DeepEqual(tt.object, tt.expected) {
				t.Errorf("%s.Default(): got %v, expected %v", tt.structName, tt.object, tt.expected)
			}
		})
	}
}
