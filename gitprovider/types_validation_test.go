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
	"fmt"
	"testing"

	"github.com/fluxcd/go-git-providers/validation"
)

type validateFunc func() error

func assertValidation(t *testing.T, structName string, validateFn validateFunc, expectedErrs []error) {
	funcName := fmt.Sprintf("%s.ValidateInfo", structName)
	wantErr := len(expectedErrs) != 0
	// Run the validation function, and make sure we expected an error (or not)
	err := validateFn()
	if (err != nil) != wantErr {
		t.Errorf("%s() error = %v, wantErr %v", funcName, err, wantErr)
	}
	// Make sure the error embeds the following expected errors
	validation.TestExpectErrors(t, funcName, err, expectedErrs...)
}

func TestDeployKey_Validate(t *testing.T) {
	tests := []struct {
		name         string
		key          DeployKeyInfo
		expectedErrs []error
	}{
		{
			name: "valid create",
			key: DeployKeyInfo{
				Name: "foo-deploykey",
				Key:  []byte("some-data"),
			},
		},
		{
			name: "valid create, with all checked fields populated",
			key: DeployKeyInfo{
				Name: "foo-deploykey",
				Key:  []byte("some-data"),
			},
		},
		{
			name: "invalid create, missing name",
			key: DeployKeyInfo{
				Key: []byte("some-data"),
			},
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name: "invalid create, missing key",
			key: DeployKeyInfo{
				Name: "foo-deploykey",
			},
			expectedErrs: []error{validation.ErrFieldRequired},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidation(t, "DeployKey", tt.key.ValidateInfo, tt.expectedErrs)
		})
	}
}

func TestRepository_Validate(t *testing.T) {
	unknownRepositoryVisibility := RepositoryVisibility("unknown")
	tests := []struct {
		name         string
		repo         RepositoryInfo
		expectedErrs []error
	}{
		{
			name: "valid create and update, with valid enum and description",
			repo: RepositoryInfo{
				Description: StringVar("foo-description"),
				Visibility:  RepositoryVisibilityVar(RepositoryVisibilityPublic),
			},
		},
		{
			name: "invalid create and update, invalid enum",
			repo: RepositoryInfo{
				Visibility: &unknownRepositoryVisibility,
			},
			expectedErrs: []error{validation.ErrFieldEnumInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidation(t, "Repository", tt.repo.ValidateInfo, tt.expectedErrs)
		})
	}
}

func TestTeamAccess_Validate(t *testing.T) {
	invalidPermission := RepositoryPermission("unknown")
	tests := []struct {
		name         string
		ta           TeamAccessInfo
		expectedErrs []error
	}{
		{
			name: "valid create, required field set",
			ta: TeamAccessInfo{
				Name: "foo-team",
			},
		},
		{
			name:         "invalid create, required name",
			ta:           TeamAccessInfo{},
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name: "valid create, also including valid repoinfo",
			ta: TeamAccessInfo{
				Name: "foo-team",
			},
		},
		{
			name: "valid create, with valid enum",
			ta: TeamAccessInfo{
				Name:       "foo-team",
				Permission: RepositoryPermissionVar(RepositoryPermissionPull),
			},
		},
		{
			name: "invalid create, invalid enum",
			ta: TeamAccessInfo{
				Name:       "foo-team",
				Permission: &invalidPermission,
			},
			expectedErrs: []error{validation.ErrFieldEnumInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertValidation(t, "TeamAccess", tt.ta.ValidateInfo, tt.expectedErrs)
		})
	}
}
