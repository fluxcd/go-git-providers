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
	"strings"
	"testing"
)

func newOrgRef(domain, org string, subOrgs []string) OrganizationInfo {
	if subOrgs == nil {
		subOrgs = []string{}
	}
	return OrganizationInfo{
		Domain:           domain,
		Organization:     org,
		SubOrganizations: subOrgs,
	}
}

func newRepoRef(domain, org string, subOrgs []string, repoName string) RepositoryInfo {
	return RepositoryInfo{
		OrganizationInfo: newOrgRef(domain, org, subOrgs),
		RepositoryName:   repoName,
	}
}

func TestParseOrganizationURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    OrganizationRef
		wantErr bool
		err     error
	}{
		{
			name: "easy",
			url:  "https://github.com/luxas",
			want: newOrgRef("github.com", "luxas", nil),
		},
		{
			name: "trailing slash",
			url:  "https://github.com/luxas/",
			want: newOrgRef("github.com", "luxas", nil),
		},
		{
			name: "one sub-org",
			url:  "https://gitlab.com/my-org/sub-org",
			want: newOrgRef("gitlab.com", "my-org", []string{"sub-org"}),
		},
		{
			name: "three sub-orgs and custom domain",
			url:  "https://my-gitlab.com:6443/my-org/sub-org/2/3",
			want: newOrgRef("my-gitlab.com:6443", "my-org", []string{"sub-org", "2", "3"}),
		},
		{
			name: "no org specified",
			url:  "https://github.com",
			err:  ErrURLInvalid,
		},
		{
			name: "no org specified, trailing slash",
			url:  "https://github.com/",
			err:  ErrURLInvalid,
		},
		{
			name: "empty parts 1",
			url:  "https://github.com/foo///",
			err:  ErrURLInvalid,
		},
		{
			name: "empty parts 2",
			url:  "https://github.com///foo///",
			err:  ErrURLInvalid,
		},
		{
			name: "empty URL",
			url:  "",
			err:  ErrURLInvalid,
		},
		{
			name: "disallow fragments",
			url:  "https://github.com/luxas#random",
			err:  ErrURLUnsupportedParts,
		},
		{
			name: "disallow query values",
			url:  "https://github.com/luxas?foo=bar",
			err:  ErrURLUnsupportedParts,
		},
		{
			name: "disallow user auth",
			url:  "https://user:pass@github.com/luxas",
			err:  ErrURLUnsupportedParts,
		},
		{
			name: "disallow http",
			url:  "http://github.com/luxas",
			err:  ErrURLUnsupportedScheme,
		},
		{
			name: "no scheme",
			url:  "github.com/luxas",
			err:  ErrURLUnsupportedScheme,
		},
		{
			name:    "invalid URL",
			url:     ":foo/bar",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOrganizationURL(tt.url)
			if tt.err != nil {
				// Infer that an error was expected if tt.err was set
				tt.wantErr = true
				// Check if we got the right error
				if !errors.Is(err, tt.err) {
					t.Errorf("ParseOrganizationURL() got error = %v, want error = %v", err, tt.err)
				}
			}
			// Check if we expected an error
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOrganizationURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Check so we have the right value
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseOrganizationURL() = %v, want %v", got, tt.want)
			}
			// Ensure a non-pointer return and that roundtrip data is preserved
			if got != nil {
				if _, ok := got.(OrganizationInfo); !ok {
					t.Error("ParseOrganizationURL(): Expected OrganizationInfo struct to be returned")
				}
				// expect the round-trip to remove any trailing slashes
				expectedURL := strings.TrimSuffix(tt.url, "/")
				if got.String() != expectedURL {
					t.Errorf("ParseOrganizationURL(): got.String() = %q, want %q", got.String(), expectedURL)
				}
			}
		})
	}
}

func TestParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    RepositoryRef
		wantErr bool
		err     error
	}{
		{
			name: "easy",
			url:  "https://github.com/luxas/foo-bar",
			want: newRepoRef("github.com", "luxas", nil, "foo-bar"),
		},
		{
			name: "trailing slash",
			url:  "https://github.com/luxas/foo-bar/",
			want: newRepoRef("github.com", "luxas", nil, "foo-bar"),
		},
		{
			name: "one sub-org",
			url:  "https://gitlab.com/my-org/sub-org/foo-bar",
			want: newRepoRef("gitlab.com", "my-org", []string{"sub-org"}, "foo-bar"),
		},
		{
			name: "three sub-orgs and custom domain",
			url:  "https://my-gitlab.com:6443/my-org/sub-org/2/3/foo-bar",
			want: newRepoRef("my-gitlab.com:6443", "my-org", []string{"sub-org", "2", "3"}, "foo-bar"),
		},
		{
			name:    "no repo specified",
			url:     "https://github.com/luxas",
			wantErr: true,
		},
		{
			name:    "no repo specified, trailing slash",
			url:     "https://github.com/luxas/",
			wantErr: true,
		},
		{
			name: "empty parts 1",
			url:  "https://github.com/luxas/foobar//",
			err:  ErrURLInvalid,
		},
		{
			name: "empty parts 2",
			url:  "https://github.com//luxas/foobar/",
			err:  ErrURLInvalid,
		},
		{
			name: "empty URL",
			url:  "",
			err:  ErrURLInvalid,
		},
		{
			name: "disallow fragments",
			url:  "https://github.com/luxas/foobar#random",
			err:  ErrURLUnsupportedParts,
		},
		{
			name: "disallow query values",
			url:  "https://github.com/luxas/foobar?foo=bar",
			err:  ErrURLUnsupportedParts,
		},
		{
			name: "disallow user auth",
			url:  "https://user:pass@github.com/luxas/foobar",
			err:  ErrURLUnsupportedParts,
		},
		{
			name: "disallow http",
			url:  "http://github.com/luxas/foobar",
			err:  ErrURLUnsupportedScheme,
		},
		{
			name: "no scheme",
			url:  "github.com/luxas/foobar",
			err:  ErrURLUnsupportedScheme,
		},
		{
			name:    "invalid URL",
			url:     ":foo/bar",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepositoryURL(tt.url)
			if tt.err != nil {
				// Infer that an error was expected if tt.err was set
				tt.wantErr = true
				// Check if we got the right error
				if !errors.Is(err, tt.err) {
					t.Errorf("ParseRepositoryURL() got error = %v, want error = %v", err, tt.err)
				}
			}
			// Check if we expected an error
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepositoryURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Check so we have the right value
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRepositoryURL() = %v, want %v", got, tt.want)
			}
			// Ensure a non-pointer return and that roundtrip data is preserved
			if got != nil {
				if _, ok := got.(RepositoryInfo); !ok {
					t.Error("ParseRepositoryURL(): Expected RepositoryInfo struct to be returned")
				}
				// expect the round-trip to remove any trailing slashes
				expectedURL := strings.TrimSuffix(tt.url, "/")
				if got.String() != expectedURL {
					t.Errorf("ParseRepositoryURL(): got.String() = %q, want %q", got.String(), expectedURL)
				}
			}
		})
	}
}

func TestGetCloneURL(t *testing.T) {
	tests := []struct {
		name      string
		repoinfo  RepositoryInfo
		transport TransportType
		want      string
	}{
		{
			name:      "https",
			repoinfo:  newRepoRef("github.com", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeHTTPS,
			want:      "https://github.com/luxas/test-org/other/foo-bar.git",
		},
		{
			name:      "git",
			repoinfo:  newRepoRef("gitlab.com", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeGit,
			want:      "git@gitlab.com:luxas/test-org/other/foo-bar.git",
		},
		{
			name:      "ssh",
			repoinfo:  newRepoRef("my-gitlab.com:6443", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeSSH,
			want:      "ssh://git@my-gitlab.com:6443/luxas/test-org/other/foo-bar",
		},
		{
			name:      "none",
			repoinfo:  newRepoRef("my-gitlab.com:6443", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportType("random"),
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1 := GetCloneURL(tt.repoinfo, tt.transport)
			if got1 != tt.want {
				t.Errorf("GetCloneURL() = %v, want %v", got1, tt.want)
			}
			got2 := tt.repoinfo.GetCloneURL(tt.transport)
			if got2 != tt.want {
				t.Errorf("RepositoryInfo.GetCloneURL() = %v, want %v", got1, tt.want)
			}
			if got1 != got2 {
				t.Errorf("GetCloneURL() = %q and RepositoryInfo.GetCloneURL() = %q should match", got1, got2)
			}
		})
	}
}
