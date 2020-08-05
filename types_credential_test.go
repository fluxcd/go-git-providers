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

func TestDeployKey_Getters(t *testing.T) {
	tests := []struct {
		name         string
		key          DeployKey
		wantReadOnly bool
	}{
		{
			name: "all fields set",
			key: DeployKey{
				InternalHolder: InternalHolder{
					Internal: "random-internal-object",
				},
				Name:       "foo",
				Key:        []byte("data"),
				ReadOnly:   BoolVar(true),
				Repository: newOrgRepoRefPtr("github.com", "foo-org", nil, "foo-repo"),
			},
			wantReadOnly: true,
		},
		{
			name:         "no fields set, use default read only",
			key:          DeployKey{},
			wantReadOnly: true,
		},
		{
			name: "respect non-default read only",
			key: DeployKey{
				ReadOnly: BoolVar(false),
			},
			wantReadOnly: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// mappings maps actual getter values to expected ones
			mappings := map[interface{}]interface{}{
				tt.key.GetInternal():      tt.key.InternalHolder.Internal,
				tt.key.GetName():          tt.key.Name,
				string(tt.key.GetData()):  string(tt.key.Key), // []byte can't be a map key
				tt.key.IsReadOnly():       tt.wantReadOnly,
				tt.key.GetRepositoryRef(): tt.key.Repository,
				tt.key.GetType():          RepositoryCredentialTypeDeployKey,
			}
			for actual, expected := range mappings {
				if !reflect.DeepEqual(actual, expected) {
					t.Errorf("Getter for DeployKey didn't return expected data. actual: %v, expected: %v", actual, expected)
				}
			}
		})
	}
}
