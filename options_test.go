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
	"errors"
	"reflect"
	"testing"

	"github.com/fluxcd/go-git-providers/validation"
)

var (
	unknownLicenseTemplate = LicenseTemplate("foo")
	repoCreateOpts1        = &RepositoryCreateOptions{AutoInit: BoolVar(true), LicenseTemplate: LicenseTemplateVar(LicenseTemplateMIT)}
	repoCreateOpts2        = &RepositoryCreateOptions{AutoInit: BoolVar(false), LicenseTemplate: LicenseTemplateVar(LicenseTemplateApache2)}
	partialCreateOpts1     = &RepositoryCreateOptions{AutoInit: BoolVar(false)}
	partialCreateOpts2     = &RepositoryCreateOptions{LicenseTemplate: LicenseTemplateVar(LicenseTemplateApache2)}
	invalidRepoCreateOpts  = &RepositoryCreateOptions{LicenseTemplate: &unknownLicenseTemplate}
)

func TestMakeRepositoryCreateOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        []RepositoryCreateOption
		want        RepositoryCreateOptions
		wantErr     bool
		expectedErr error
	}{
		{
			name: "default nil pointers",
			want: RepositoryCreateOptions{},
		},
		{
			name: "set all fields",
			opts: []RepositoryCreateOption{repoCreateOpts1},
			want: *repoCreateOpts1,
		},
		{
			name: "latter overrides former",
			opts: []RepositoryCreateOption{
				repoCreateOpts1,
				repoCreateOpts2,
			},
			want: *repoCreateOpts2,
		},
		{
			name:        "invalid license template",
			opts:        []RepositoryCreateOption{invalidRepoCreateOpts},
			want:        *invalidRepoCreateOpts,
			expectedErr: validation.ErrFieldEnumInvalid,
		},
		{
			name: "partial options can form an unit",
			opts: []RepositoryCreateOption{
				partialCreateOpts1,
				partialCreateOpts2,
			},
			want: *repoCreateOpts2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeRepositoryCreateOptions(tt.opts...)
			if tt.expectedErr != nil {
				tt.wantErr = true // infer that an error is wanted
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("MakeRepositoryCreateOptions() error = %v, wanted %v", err, tt.expectedErr)
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeRepositoryCreateOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeRepositoryCreateOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
