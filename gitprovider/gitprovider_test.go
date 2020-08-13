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

func TestValidateAndDefaultInfo(t *testing.T) {
	tests := []struct {
		name     string
		info     CreatableInfo
		expected CreatableInfo
		wantErr  bool
	}{
		{
			name: "valid => defaulting",
			info: &TeamAccessInfo{
				Name: "foo",
			},
			expected: &TeamAccessInfo{
				Name:       "foo",
				Permission: RepositoryPermissionVar(defaultRepoPermission),
			},
			wantErr: false,
		},
		{
			name:     "invalid => no defaulting + error",
			info:     &TeamAccessInfo{},
			expected: &TeamAccessInfo{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateAndDefaultInfo(tt.info); (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndDefaultInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.info, tt.expected) {
				t.Errorf("ValidateAndDefaultInfo() object = %v, wanted %v", tt.info, tt.expected)
			}
		})
	}
}
