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
		object     Creatable
		expected   Creatable
	}{
		{
			name:       "DeployKey: empty",
			structName: "DeployKey",
			object:     &DeployKey{},
			expected: &DeployKey{
				ReadOnly: BoolVar(true),
			},
		},
		{
			name:       "DeployKey: don't set if non-nil (default)",
			structName: "DeployKey",
			object: &DeployKey{
				ReadOnly: BoolVar(true),
			},
			expected: &DeployKey{
				ReadOnly: BoolVar(true),
			},
		},
		{
			name:       "DeployKey: don't set if non-nil (non-default)",
			structName: "DeployKey",
			object: &DeployKey{
				ReadOnly: BoolVar(false),
			},
			expected: &DeployKey{
				ReadOnly: BoolVar(false),
			},
		},
		{
			name:       "Repository: empty",
			structName: "Repository",
			object:     &Repository{},
			expected: &Repository{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityPrivate),
				DefaultBranch: StringVar("master"),
			},
		},
		{
			name:       "Repository: don't set if non-nil (default)",
			structName: "Repository",
			object: &Repository{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityPrivate),
				DefaultBranch: StringVar("master"),
			},
			expected: &Repository{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityPrivate),
				DefaultBranch: StringVar("master"),
			},
		},
		{
			name:       "Repository: don't set if non-nil (non-default)",
			structName: "Repository",
			object: &Repository{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityInternal),
				DefaultBranch: StringVar("main"),
			},
			expected: &Repository{
				Visibility:    RepositoryVisibilityVar(RepositoryVisibilityInternal),
				DefaultBranch: StringVar("main"),
			},
		},
		{
			name:       "TeamAccess: empty",
			structName: "TeamAccess",
			object:     &TeamAccess{},
			expected: &TeamAccess{
				Permission: RepositoryPermissionVar(RepositoryPermissionPull),
			},
		},
		{
			name:       "TeamAccess: don't set if non-nil (default)",
			structName: "Repository",
			object: &TeamAccess{
				Permission: RepositoryPermissionVar(RepositoryPermissionPull),
			},
			expected: &TeamAccess{
				Permission: RepositoryPermissionVar(RepositoryPermissionPull),
			},
		},
		{
			name:       "TeamAccess: don't set if non-nil (non-default)",
			structName: "TeamAccess",
			object: &TeamAccess{
				Permission: RepositoryPermissionVar(RepositoryPermissionPush),
			},
			expected: &TeamAccess{
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
