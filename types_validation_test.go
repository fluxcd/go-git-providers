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
)

type validateMethod string

const (
	validateCreate = validateMethod("Create")
	validateUpdate = validateMethod("Update")
	validateDelete = validateMethod("Delete")
)

type validateFunc func() error

func assertValidation(t *testing.T, structName string, method validateMethod, validateFn validateFunc, expectedErrs []error) {
	funcName := fmt.Sprintf("%s.Validate%s", structName, method)
	wantErr := len(expectedErrs) != 0
	// Run the validation function, and make sure we expected an error (or not)
	err := validateFn()
	if (err != nil) != wantErr {
		t.Errorf("%s() error = %v, wantErr %v", funcName, err, wantErr)
	}
	// Make sure the error embeds the following expected errors
	expectErrors(t, funcName, err, expectedErrs)
}

func TestDeployKey_Validate(t *testing.T) {
	tests := []struct {
		name         string
		key          DeployKey
		methods      []validateMethod
		expectedErrs []error
	}{
		{
			name: "valid create",
			key: DeployKey{
				Name: "foo-deploykey",
				Key:  []byte("some-data"),
			},
			methods: []validateMethod{validateCreate},
		},
		{
			name: "valid delete",
			key: DeployKey{
				Name: "foo-deploykey",
				Key:  []byte("some-data"),
			},
			methods: []validateMethod{validateDelete},
		},
		{
			name: "valid create, with all checked fields populated",
			key: DeployKey{
				Name:       "foo-deploykey",
				Key:        []byte("some-data"),
				Repository: newRepoInfoPtr("github.com", "foo-org", nil, "foo-repo"),
			},
			methods: []validateMethod{validateCreate},
		},
		{
			name: "valid delete, with all checked fields populated",
			key: DeployKey{
				Name:       "foo-deploykey",
				Repository: newRepoInfoPtr("github.com", "foo-org", nil, "foo-repo"),
			},
			methods: []validateMethod{validateDelete},
		},
		{
			name: "invalid create, missing name",
			key: DeployKey{
				Key: []byte("some-data"),
			},
			expectedErrs: []error{ErrFieldRequired},
			methods:      []validateMethod{validateCreate},
		},
		{
			name:         "invalid delete, missing name",
			key:          DeployKey{},
			expectedErrs: []error{ErrFieldRequired},
			methods:      []validateMethod{validateDelete},
		},
		{
			name: "invalid create, missing key",
			key: DeployKey{
				Name: "foo-deploykey",
			},
			expectedErrs: []error{ErrFieldRequired},
			methods:      []validateMethod{validateCreate},
		},
		{
			name: "invalid create, invalid org info",
			key: DeployKey{
				Name:       "foo-deploykey",
				Key:        []byte("some-data"),
				Repository: newRepoInfoPtr("github.com", "", nil, "foo-repo"),
			},
			expectedErrs: []error{ErrFieldRequired},
			methods:      []validateMethod{validateCreate},
		},
		{
			name: "invalid delete, invalid org info",
			key: DeployKey{
				Name:       "foo-deploykey",
				Repository: newRepoInfoPtr("github.com", "", nil, "foo-repo"),
			},
			expectedErrs: []error{ErrFieldRequired},
			methods:      []validateMethod{validateDelete},
		},
		{
			name: "invalid create, invalid repo info",
			key: DeployKey{
				Name:       "foo-deploykey",
				Key:        []byte("some-data"),
				Repository: newRepoInfoPtr("github.com", "foo-org", nil, ""),
			},
			expectedErrs: []error{ErrFieldRequired},
			methods:      []validateMethod{validateCreate},
		},
		{
			name: "invalid delete, invalid repo info",
			key: DeployKey{
				Name:       "foo-deploykey",
				Repository: newRepoInfoPtr("github.com", "foo-org", nil, ""),
			},
			expectedErrs: []error{ErrFieldRequired},
			methods:      []validateMethod{validateDelete},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range tt.methods {
				var validateFn validateFunc
				switch method {
				case validateCreate:
					validateFn = tt.key.ValidateCreate
				case validateDelete:
					validateFn = tt.key.ValidateDelete
				default:
					t.Errorf("unknown validate method: %s", method)
					return
				}

				assertValidation(t, "DeployKey", method, validateFn, tt.expectedErrs)
			}
		})
	}
}

func TestRepository_Validate(t *testing.T) {
	unknownRepoVisibility := RepoVisibility("unknown")
	tests := []struct {
		name         string
		repo         Repository
		methods      []validateMethod
		expectedErrs []error
	}{
		{
			name: "valid create and update, without enums",
			repo: Repository{
				RepositoryInfo: newRepoInfo("github.com", "foo-org", nil, "foo-repo"),
			},
			methods: []validateMethod{validateCreate, validateUpdate},
		},
		{
			name: "valid create and update, with valid enum and description",
			repo: Repository{
				RepositoryInfo: newRepoInfo("github.com", "foo-org", nil, "foo-repo"),
				Description:    stringVar("foo-description"),
				Visibility:     repoVisibilityVar(RepoVisibilityPublic),
			},
			methods: []validateMethod{validateCreate, validateUpdate},
		},
		{
			name: "invalid create and update, invalid enum",
			repo: Repository{
				RepositoryInfo: newRepoInfo("github.com", "foo-org", nil, "foo-repo"),
				Visibility:     &unknownRepoVisibility,
			},
			methods:      []validateMethod{validateCreate, validateUpdate},
			expectedErrs: []error{ErrFieldEnumInvalid},
		},
		{
			name: "invalid create and update, invalid repo info",
			repo: Repository{
				RepositoryInfo: newRepoInfo("github.com", "foo-org", nil, ""),
				Visibility:     repoVisibilityVar(RepoVisibilityPrivate),
			},
			methods:      []validateMethod{validateCreate, validateUpdate},
			expectedErrs: []error{ErrFieldRequired},
		},
		{
			name: "invalid create and update, invalid org info",
			repo: Repository{
				RepositoryInfo: newRepoInfo("github.com", "", nil, "foo-repo"), // invalid org name
				Description:    stringVar(""),                                  // description isn't validated, doesn't need any for now
			},
			methods:      []validateMethod{validateCreate, validateUpdate},
			expectedErrs: []error{ErrFieldRequired},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range tt.methods {
				var validateFn validateFunc
				switch method {
				case validateCreate:
					validateFn = tt.repo.ValidateCreate
				case validateUpdate:
					validateFn = tt.repo.ValidateUpdate
				default:
					t.Errorf("unknown validate method: %s", method)
					return
				}

				assertValidation(t, "Repository", method, validateFn, tt.expectedErrs)
			}
		})
	}
}

func TestTeamAccess_Validate(t *testing.T) {
	invalidPermission := RepositoryPermission("unknown")
	tests := []struct {
		name         string
		ta           TeamAccess
		methods      []validateMethod
		expectedErrs []error
	}{
		{
			name: "valid create and delete, required field set",
			ta: TeamAccess{
				Name: "foo-team",
			},
			methods: []validateMethod{validateCreate, validateDelete},
		},
		{
			name:         "invalid create and delete, required name",
			ta:           TeamAccess{},
			methods:      []validateMethod{validateCreate, validateDelete},
			expectedErrs: []error{ErrFieldRequired},
		},
		{
			name: "valid create and delete, also including valid repoinfo",
			ta: TeamAccess{
				Name:       "foo-team",
				Repository: newRepoInfoPtr("github.com", "foo-org", nil, "foo-repo"),
			},
			methods: []validateMethod{validateCreate, validateDelete},
		},
		{
			name: "invalid create and delete, invalid repoinfo",
			ta: TeamAccess{
				Name:       "foo-team",
				Repository: newRepoInfoPtr("github.com", "foo-org", nil, ""),
			},
			methods:      []validateMethod{validateCreate, validateDelete},
			expectedErrs: []error{ErrFieldRequired},
		},
		{
			name: "invalid create and delete, invalid orginfo",
			ta: TeamAccess{
				Name:       "foo-team",
				Repository: newRepoInfoPtr("", "foo-org", nil, "foo-repo"),
			},
			methods:      []validateMethod{validateCreate, validateDelete},
			expectedErrs: []error{ErrFieldRequired},
		},
		{
			name: "valid create, with valid enum",
			ta: TeamAccess{
				Name:       "foo-team",
				Permission: repositoryPermissionVar(RepositoryPermissionPull),
			},
			methods: []validateMethod{validateCreate},
		},
		{
			name: "invalid create, invalid enum",
			ta: TeamAccess{
				Name:       "foo-team",
				Permission: &invalidPermission,
			},
			methods:      []validateMethod{validateCreate},
			expectedErrs: []error{ErrFieldEnumInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range tt.methods {
				var validateFn validateFunc
				switch method {
				case validateCreate:
					validateFn = tt.ta.ValidateCreate
				case validateDelete:
					validateFn = tt.ta.ValidateDelete
				default:
					t.Errorf("unknown validate method: %s", method)
					return
				}

				assertValidation(t, "TeamAccess", method, validateFn, tt.expectedErrs)
			}
		})
	}
}
