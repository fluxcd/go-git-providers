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
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/fluxcd/go-git-providers/validation"
)

func newOrgRef(domain, org string, subOrgs []string) OrganizationRef {
	if subOrgs == nil {
		subOrgs = []string{}
	}
	return OrganizationRef{
		Domain:           domain,
		Organization:     org,
		SubOrganizations: subOrgs,
	}
}

func newOrgRefPtr(domain, org string, subOrgs []string) *OrganizationRef {
	orgRef := newOrgRef(domain, org, subOrgs)
	return &orgRef
}

func newOrgRepoRef(domain, org string, subOrgs []string, repoName string) OrgRepositoryRef {
	return OrgRepositoryRef{
		OrganizationRef: newOrgRef(domain, org, subOrgs),
		RepositoryName:  repoName,
	}
}

func newOrgRepoRefPtr(domain, org string, subOrgs []string, repoName string) *OrgRepositoryRef {
	repoInfo := newOrgRepoRef(domain, org, subOrgs, repoName)
	return &repoInfo
}

func newUserRef(domain, userLogin string) UserRef {
	return UserRef{
		Domain:    domain,
		UserLogin: userLogin,
	}
}

func newUserRefPtr(domain, userLogin string) *UserRef {
	userRef := newUserRef(domain, userLogin)
	return &userRef
}

func newUserRepoRef(domain, userLogin, repoName string) UserRepositoryRef {
	return UserRepositoryRef{
		UserRef:        newUserRef(domain, userLogin),
		RepositoryName: repoName,
	}
}

func newUserRepoRefPtr(domain, userLogin, repoName string) *UserRepositoryRef {
	repoInfo := newUserRepoRef(domain, userLogin, repoName)
	return &repoInfo
}

func TestParseOrganizationURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want *OrganizationRef
		err  error
	}{
		{
			name: "easy",
			url:  "https://github.com/my-org",
			want: newOrgRefPtr("github.com", "my-org", nil),
		},
		{
			name: "trailing slash",
			url:  "https://github.com/my-org/",
			want: newOrgRefPtr("github.com", "my-org", nil),
		},
		{
			name: "one sub-org",
			url:  "https://gitlab.com/my-org/sub-org",
			want: newOrgRefPtr("gitlab.com", "my-org", []string{"sub-org"}),
		},
		{
			name: "three sub-orgs and custom domain",
			url:  "https://my-gitlab.com:6443/my-org/sub-org/2/3",
			want: newOrgRefPtr("my-gitlab.com:6443", "my-org", []string{"sub-org", "2", "3"}),
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
			name: "invalid URL",
			url:  ":foo/bar",
			err:  &url.Error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOrganizationURL(tt.url)
			// Validate so that the error is expected
			validation.TestExpectErrors(t, "ParseOrganizationURL", err, tt.err)
			// Check so we have the right value
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseOrganizationURL() = %v, want %v", got, tt.want)
			}
			// Ensure that roundtrip data is preserved
			if got != nil {
				// expect the round-trip to remove any trailing slashes
				expectedURL := strings.TrimSuffix(tt.url, "/")
				if got.String() != expectedURL {
					t.Errorf("ParseOrganizationURL(): got.String() = %q, want %q", got.String(), expectedURL)
				}
			}
		})
	}
}

func TestParseUserURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want *UserRef
		err  error
	}{
		{
			name: "easy",
			url:  "https://github.com/my-user",
			want: newUserRefPtr("github.com", "my-user"),
		},
		{
			name: "trailing slash",
			url:  "https://github.com/my-user/",
			want: newUserRefPtr("github.com", "my-user"),
		},
		{
			name: "custom domain",
			url:  "https://my-gitlab.com:6443/my-user/",
			want: newUserRefPtr("my-gitlab.com:6443", "my-user"),
		},
		{
			name: "can't have sub-orgs",
			url:  "https://my-gitlab.com:6443/my-user/my-sub-org",
			err:  ErrURLInvalid,
		},
		{
			name: "no user specified",
			url:  "https://github.com",
			err:  ErrURLInvalid,
		},
		{
			name: "no user specified, trailing slash",
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
			name: "invalid URL",
			url:  ":foo/bar",
			err:  &url.Error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUserURL(tt.url)
			// Validate so that the error is expected
			validation.TestExpectErrors(t, "ParseUserURL", err, tt.err)
			// Check so we have the right value
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseUserURL() = %v, want %v", got, tt.want)
			}
			// Ensure that roundtrip data is preserved
			if got != nil {
				// expect the round-trip to remove any trailing slashes
				expectedURL := strings.TrimSuffix(tt.url, "/")
				if got.String() != expectedURL {
					t.Errorf("ParseUserURL(): got.String() = %q, want %q", got.String(), expectedURL)
				}
			}
		})
	}
}

func TestParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		types    []IdentityType
		wantOrg  *OrgRepositoryRef
		wantUser *UserRepositoryRef
		err      error // expected error
	}{
		{
			name:     "easy",
			url:      "https://github.com/identity/foo-bar",
			types:    []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			wantUser: newUserRepoRefPtr("github.com", "identity", "foo-bar"),
			wantOrg:  newOrgRepoRefPtr("github.com", "identity", nil, "foo-bar"),
		},
		{
			name:     "trailing slash",
			url:      "https://github.com/identity/foo-bar/",
			types:    []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			wantUser: newUserRepoRefPtr("github.com", "identity", "foo-bar"),
			wantOrg:  newOrgRepoRefPtr("github.com", "identity", nil, "foo-bar"),
		},
		{
			name:     "including a dot",
			url:      "https://github.com/identity/foo-bar.withdot",
			types:    []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			wantUser: newUserRepoRefPtr("github.com", "identity", "foo-bar.withdot"),
			wantOrg:  newOrgRepoRefPtr("github.com", "identity", nil, "foo-bar.withdot"),
		},
		{
			name:     "strip git suffix",
			url:      "https://github.com/identity/foo-bar.git",
			types:    []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			wantUser: newUserRepoRefPtr("github.com", "identity", "foo-bar"),
			wantOrg:  newOrgRepoRefPtr("github.com", "identity", nil, "foo-bar"),
		},
		{
			name:  "user, one sub-org",
			url:   "https://gitlab.com/my-org/sub-org/foo-bar",
			types: []IdentityType{IdentityTypeUser},
			err:   ErrURLInvalid,
		},
		{
			name:    "organization, one sub-org",
			url:     "https://gitlab.com/my-org/sub-org/foo-bar",
			types:   []IdentityType{IdentityTypeOrganization},
			wantOrg: newOrgRepoRefPtr("gitlab.com", "my-org", []string{"sub-org"}, "foo-bar"),
		},
		{
			name:  "user, three sub-orgs and custom domain",
			url:   "https://my-gitlab.com:6443/my-org/sub-org/2/3/foo-bar",
			types: []IdentityType{IdentityTypeUser},
			err:   ErrURLInvalid,
		},
		{
			name:    "organization, three sub-orgs and custom domain",
			url:     "https://my-gitlab.com:6443/my-org/sub-org/2/3/foo-bar",
			types:   []IdentityType{IdentityTypeOrganization},
			wantOrg: newOrgRepoRefPtr("my-gitlab.com:6443", "my-org", []string{"sub-org", "2", "3"}, "foo-bar"),
		},
		{
			name:  "no repo specified",
			url:   "https://github.com/luxas",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLMissingRepoName,
		},
		{
			name:  "no repo specified, trailing slash",
			url:   "https://github.com/luxas/",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLMissingRepoName,
		},
		{
			name:  "empty parts 1",
			url:   "https://github.com/luxas/foobar//",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLInvalid,
		},
		{
			name:  "empty parts 2",
			url:   "https://github.com//luxas/foobar/",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLInvalid,
		},
		{
			name:  "empty URL",
			url:   "",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLInvalid,
		},
		{
			name:  "disallow fragments",
			url:   "https://github.com/luxas/foobar#random",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLUnsupportedParts,
		},
		{
			name:  "disallow query values",
			url:   "https://github.com/luxas/foobar?foo=bar",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLUnsupportedParts,
		},
		{
			name:  "disallow user auth",
			url:   "https://user:pass@github.com/luxas/foobar",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLUnsupportedParts,
		},
		{
			name:  "disallow http",
			url:   "http://github.com/luxas/foobar",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLUnsupportedScheme,
		},
		{
			name:  "no scheme",
			url:   "github.com/luxas/foobar",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   ErrURLUnsupportedScheme,
		},
		{
			name:  "invalid URL",
			url:   ":foo/bar",
			types: []IdentityType{IdentityTypeOrganization, IdentityTypeUser},
			err:   &url.Error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.types) == 0 {
				t.Fatal("must set tt.types to one or more values")
			}
			for _, identityType := range tt.types {
				var err error
				var resultStr string
				switch identityType {
				case IdentityTypeUser:
					var res *UserRepositoryRef
					res, err = ParseUserRepositoryURL(tt.url)
					// Check so we have the right value
					if !reflect.DeepEqual(res, tt.wantUser) {
						t.Errorf("ParseUserRepositoryURL() = %v, want %v", res, tt.wantUser)
					}
					if res != nil {
						resultStr = res.String()
					}
				case IdentityTypeOrganization:
					var res *OrgRepositoryRef
					res, err = ParseOrgRepositoryURL(tt.url)
					// Check so we have the right value
					if !reflect.DeepEqual(res, tt.wantOrg) {
						t.Errorf("ParseOrgRepositoryURL() = %v, want %v", res, tt.wantOrg)
					}
					if res != nil {
						resultStr = res.String()
					}
				default:
					t.Fatalf("invalid identityType: %v", identityType)
				}

				// Validate so that the error is expected
				validation.TestExpectErrors(t, "ParseRepositoryURL", err, tt.err)

				// Ensure that roundtrip data is preserved
				if resultStr != "" {
					// expect the round-trip to remove any trailing slashes
					expectedURL := strings.TrimSuffix(tt.url, "/")
					// expect any .git suffix to be removed
					expectedURL = strings.TrimSuffix(expectedURL, ".git")
					if resultStr != expectedURL {
						t.Errorf("Parse{Org,User}RepositoryURL(): resultStr = %q, want %q", resultStr, expectedURL)
					}
				}
			}
		})
	}
}

func TestGetCloneURL(t *testing.T) {
	tests := []struct {
		name      string
		repoinfo  RepositoryRef
		transport TransportType
		want      string
	}{
		{
			name:      "org: https",
			repoinfo:  newOrgRepoRef("github.com", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeHTTPS,
			want:      "https://github.com/luxas/test-org/other/foo-bar.git",
		},
		{
			name:      "org: git",
			repoinfo:  newOrgRepoRef("gitlab.com", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeGit,
			want:      "git@gitlab.com:luxas/test-org/other/foo-bar.git",
		},
		{
			name:      "org: ssh",
			repoinfo:  newOrgRepoRef("my-gitlab.com:6443", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeSSH,
			want:      "ssh://git@my-gitlab.com:6443/luxas/test-org/other/foo-bar",
		},
		{
			name:      "org: none",
			repoinfo:  newOrgRepoRef("my-gitlab.com:6443", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportType("random"),
			want:      "",
		},
		{
			name:      "user: https",
			repoinfo:  newUserRepoRef("github.com", "luxas", "foo-bar"),
			transport: TransportTypeHTTPS,
			want:      "https://github.com/luxas/foo-bar.git",
		},
		{
			name:      "user: git",
			repoinfo:  newUserRepoRef("gitlab.com", "luxas", "foo-bar"),
			transport: TransportTypeGit,
			want:      "git@gitlab.com:luxas/foo-bar.git",
		},
		{
			name:      "user: ssh",
			repoinfo:  newUserRepoRef("my-gitlab.com:6443", "luxas", "foo-bar"),
			transport: TransportTypeSSH,
			want:      "ssh://git@my-gitlab.com:6443/luxas/foo-bar",
		},
		{
			name:      "user: none",
			repoinfo:  newUserRepoRef("my-gitlab.com:6443", "luxas", "foo-bar"),
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
				t.Errorf("RepositoryRef.GetCloneURL() = %v, want %v", got1, tt.want)
			}
			if got1 != got2 {
				t.Errorf("GetCloneURL() = %q and RepositoryRef.GetCloneURL() = %q should match", got1, got2)
			}
		})
	}
}

func TestIdentityRef_GetType(t *testing.T) {
	tests := []struct {
		name string
		ref  IdentityRef
		want IdentityType
	}{
		{
			name: "sample user",
			ref:  newUserRef("github.com", "bar"),
			want: IdentityTypeUser,
		},
		{
			name: "sample top-level org",
			ref:  newOrgRef("github.com", "bar", nil),
			want: IdentityTypeOrganization,
		},
		{
			name: "sample sub-org",
			ref:  newOrgRef("github.com", "bar", []string{"baz"}),
			want: IdentityTypeSuborganization,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ref.GetType(); got != tt.want {
				t.Errorf("IdentityRef.GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepositoryRef_ValidateFields(t *testing.T) {
	tests := []struct {
		name         string
		ref          RepositoryRef
		expectedErrs []error
	}{
		{
			name: "valid user",
			ref:  newUserRepoRef("github.com", "my-user", "my-repo"),
		},
		{
			name: "valid org",
			ref:  newOrgRepoRef("github.com", "my-org", nil, "my-repo"),
		},
		{
			name: "valid sub-org",
			ref:  newOrgRepoRef("github.com", "my-org", []string{"sub-org"}, "my-repo"),
		},
		{
			name:         "missing user reponame",
			ref:          newUserRepoRef("my-gitlab.com:6443", "my-user", ""),
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name:         "missing user login",
			ref:          newUserRepoRef("my-gitlab.com:6443", "", "my-repo"),
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name:         "missing user login",
			ref:          newUserRepoRef("my-gitlab.com:6443", "", "my-repo"),
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name:         "missing user domain",
			ref:          newUserRepoRef("", "my-user", "my-repo"),
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name:         "missing org reponame",
			ref:          newOrgRepoRef("my-gitlab.com:6443", "my-org", nil, ""),
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name:         "missing org name",
			ref:          newOrgRepoRef("my-gitlab.com:6443", "", []string{"sub-org"}, "my-repo"),
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name:         "missing org domain",
			ref:          newOrgRepoRef("", "my-org", []string{"sub-org"}, "my-repo"),
			expectedErrs: []error{validation.ErrFieldRequired},
		},
		{
			name:         "multiple errors",
			ref:          newOrgRepoRef("", "", []string{"sub-org"}, "my-repo"),
			expectedErrs: []error{validation.ErrFieldRequired, &validation.MultiError{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := validation.New("test")
			tt.ref.ValidateFields(validator)
			err := validator.Error()
			validation.TestExpectErrors(t, "{User,Org}RepositoryRef.ValidateFields", err, tt.expectedErrs...)
		})
	}
}
