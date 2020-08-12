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

	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/fluxcd/go-git-providers/validation"
)

func Test_getPermissionFromMap(t *testing.T) {
	tests := []struct {
		name        string
		permissions map[string]bool
		want        *gitprovider.RepositoryPermission
	}{
		{
			name: "pull",
			permissions: map[string]bool{
				"pull":     true,
				"triage":   false,
				"push":     false,
				"maintain": false,
				"admin":    false,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPull),
		},
		{
			name: "push",
			permissions: map[string]bool{
				"pull":     true,
				"triage":   false,
				"push":     true,
				"maintain": false,
				"admin":    false,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionPush),
		},
		{
			name: "admin",
			permissions: map[string]bool{
				"pull":     true,
				"triage":   true,
				"push":     true,
				"maintain": true,
				"admin":    true,
			},
			want: gitprovider.RepositoryPermissionVar(gitprovider.RepositoryPermissionAdmin),
		},
		{
			name: "none",
			permissions: map[string]bool{
				"pull":     false,
				"triage":   false,
				"push":     false,
				"maintain": false,
				"admin":    false,
			},
			want: nil,
		},
		{
			name: "false data",
			permissions: map[string]bool{
				"pull":     false,
				"triage":   false,
				"push":     false,
				"maintain": false,
				"admin":    false,
				"invalid":  true,
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

func Test_validateAPIObject(t *testing.T) {
	tests := []struct {
		name         string
		structName   string
		fn           func(validation.Validator)
		expectedErrs []error
	}{
		{
			name:       "no error => nil",
			structName: "Foo",
			fn:         func(validation.Validator) {},
		},
		{
			name:       "one error => MultiError & InvalidServerData",
			structName: "Foo",
			fn: func(v validation.Validator) {
				v.Required("FieldBar")
			},
			expectedErrs: []error{gitprovider.ErrInvalidServerData, &validation.MultiError{}, validation.ErrFieldRequired},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIObject(tt.structName, tt.fn)
			validation.TestExpectErrors(t, "validateAPIObject", err, tt.expectedErrs...)
		})
	}
}
