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

var (
	licenseTemplateMIT     = LicenseTemplateMIT
	licenseTemplateApache2 = LicenseTemplateApache2
	licenseTemplateFoo     = LicenseTemplate("foo")
	repoCreateOpts1        = RepositoryCreateOptions{AutoInit: boolVar(true), LicenseTemplate: &licenseTemplateMIT}
	repoCreateOpts2        = RepositoryCreateOptions{AutoInit: boolVar(false), LicenseTemplate: &licenseTemplateApache2}
	invalidRepoCreateOpts  = RepositoryCreateOptions{LicenseTemplate: &licenseTemplateFoo}
)

func TestMakeRepositoryCreateOptions(t *testing.T) {
	tests := []struct {
		name    string
		fns     []RepositoryCreateOptionsFunc
		want    RepositoryCreateOptions
		wantErr bool
	}{
		{
			name:    "default nil pointers",
			want:    RepositoryCreateOptions{},
			wantErr: false,
		},
		{
			name:    "set all fields",
			fns:     []RepositoryCreateOptionsFunc{WithRepositoryCreateOptions(repoCreateOpts1)},
			want:    repoCreateOpts1,
			wantErr: false,
		},
		{
			name: "latter overrides former",
			fns: []RepositoryCreateOptionsFunc{
				WithRepositoryCreateOptions(repoCreateOpts1),
				WithRepositoryCreateOptions(repoCreateOpts2),
			},
			want:    repoCreateOpts2,
			wantErr: false,
		},
		{
			name:    "invalid license template",
			fns:     []RepositoryCreateOptionsFunc{WithRepositoryCreateOptions(invalidRepoCreateOpts)},
			want:    invalidRepoCreateOpts,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeRepositoryCreateOptions(tt.fns...)
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
