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

func newOrgInfo(domain, org string, subOrgs []string) OrganizationInfo {
	if subOrgs == nil {
		subOrgs = []string{}
	}
	return OrganizationInfo{
		Domain:           domain,
		Organization:     org,
		SubOrganizations: subOrgs,
	}
}

func newOrgInfoPtr(domain, org string, subOrgs []string) *OrganizationInfo {
	orgInfo := newOrgInfo(domain, org, subOrgs)
	return &orgInfo
}

func newOrgRepoInfo(domain, org string, subOrgs []string, repoName string) RepositoryInfo {
	return RepositoryInfo{
		IdentityRef:    newOrgInfo(domain, org, subOrgs),
		RepositoryName: repoName,
	}
}

func newOrgRepoInfoPtr(domain, org string, subOrgs []string, repoName string) *RepositoryInfo {
	repoInfo := newOrgRepoInfo(domain, org, subOrgs, repoName)
	return &repoInfo
}

func newUserInfo(domain, userLogin string) UserInfo {
	return UserInfo{
		Domain:    domain,
		UserLogin: userLogin,
	}
}

func newUserInfoPtr(domain, userLogin string) *UserInfo {
	userInfo := newUserInfo(domain, userLogin)
	return &userInfo
}

func newUserRepoInfo(domain, userLogin, repoName string) RepositoryInfo {
	return RepositoryInfo{
		IdentityRef:    newUserInfo(domain, userLogin),
		RepositoryName: repoName,
	}
}

func newUserRepoInfoPtr(domain, userLogin, repoName string) *RepositoryInfo {
	repoInfo := newUserRepoInfo(domain, userLogin, repoName)
	return &repoInfo
}

func TestParseOrganizationURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want OrganizationRef
		err  error
	}{
		{
			name: "easy",
			url:  "https://github.com/my-org",
			want: newOrgInfo("github.com", "my-org", nil),
		},
		{
			name: "trailing slash",
			url:  "https://github.com/my-org/",
			want: newOrgInfo("github.com", "my-org", nil),
		},
		{
			name: "one sub-org",
			url:  "https://gitlab.com/my-org/sub-org",
			want: newOrgInfo("gitlab.com", "my-org", []string{"sub-org"}),
		},
		{
			name: "three sub-orgs and custom domain",
			url:  "https://my-gitlab.com:6443/my-org/sub-org/2/3",
			want: newOrgInfo("my-gitlab.com:6443", "my-org", []string{"sub-org", "2", "3"}),
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

func TestParseUserURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want UserRef
		err  error
	}{
		{
			name: "easy",
			url:  "https://github.com/my-user",
			want: newUserInfo("github.com", "my-user"),
		},
		{
			name: "trailing slash",
			url:  "https://github.com/my-user/",
			want: newUserInfo("github.com", "my-user"),
		},
		{
			name: "custom domain",
			url:  "https://my-gitlab.com:6443/my-user/",
			want: newUserInfo("my-gitlab.com:6443", "my-user"),
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
			// Ensure a non-pointer return and that roundtrip data is preserved
			if got != nil {
				if _, ok := got.(UserInfo); !ok {
					t.Error("ParseUserURL(): Expected UserInfo struct to be returned")
				}
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
		name  string
		url   string
		isOrg []bool
		want  RepositoryRef
		err   error // expected error
	}{
		{
			name:  "easy user",
			url:   "https://github.com/luxas/foo-bar",
			isOrg: []bool{false},
			want:  newUserRepoInfo("github.com", "luxas", "foo-bar"),
		},
		{
			name:  "easy organization",
			url:   "https://github.com/my-org/foo-bar",
			isOrg: []bool{true},
			want:  newOrgRepoInfo("github.com", "my-org", nil, "foo-bar"),
		},
		{
			name:  "user, trailing slash",
			url:   "https://github.com/luxas/foo-bar/",
			isOrg: []bool{false},
			want:  newUserRepoInfo("github.com", "luxas", "foo-bar"),
		},
		{
			name:  "organization, trailing slash",
			url:   "https://github.com/my-org/foo-bar/",
			isOrg: []bool{true},
			want:  newOrgRepoInfo("github.com", "my-org", nil, "foo-bar"),
		},
		{
			name:  "user, including a dot",
			url:   "https://github.com/luxas/foo-bar.withdot",
			isOrg: []bool{false},
			want:  newUserRepoInfo("github.com", "luxas", "foo-bar.withdot"),
		},
		{
			name:  "organization, including a dot",
			url:   "https://github.com/my-org/foo-bar.withdot",
			isOrg: []bool{true},
			want:  newOrgRepoInfo("github.com", "my-org", nil, "foo-bar.withdot"),
		},
		{
			name:  "user, strip git suffix",
			url:   "https://github.com/luxas/foo-bar.git",
			isOrg: []bool{false},
			want:  newUserRepoInfo("github.com", "luxas", "foo-bar"),
		},
		{
			name:  "organization, strip git suffix",
			url:   "https://github.com/my-org/foo-bar.git",
			isOrg: []bool{true},
			want:  newOrgRepoInfo("github.com", "my-org", nil, "foo-bar"),
		},
		{
			name:  "user, one sub-org",
			url:   "https://gitlab.com/my-org/sub-org/foo-bar",
			isOrg: []bool{false},
			err:   ErrURLInvalid,
		},
		{
			name:  "organization, one sub-org",
			url:   "https://gitlab.com/my-org/sub-org/foo-bar",
			isOrg: []bool{true},
			want:  newOrgRepoInfo("gitlab.com", "my-org", []string{"sub-org"}, "foo-bar"),
		},
		{
			name:  "user, three sub-orgs and custom domain",
			url:   "https://my-gitlab.com:6443/my-org/sub-org/2/3/foo-bar",
			isOrg: []bool{false},
			err:   ErrURLInvalid,
		},
		{
			name:  "organization, three sub-orgs and custom domain",
			url:   "https://my-gitlab.com:6443/my-org/sub-org/2/3/foo-bar",
			isOrg: []bool{true},
			want:  newOrgRepoInfo("my-gitlab.com:6443", "my-org", []string{"sub-org", "2", "3"}, "foo-bar"),
		},
		{
			name:  "no repo specified",
			url:   "https://github.com/luxas",
			isOrg: []bool{true, false},
			err:   ErrURLMissingRepoName,
		},
		{
			name:  "no repo specified, trailing slash",
			url:   "https://github.com/luxas/",
			isOrg: []bool{true, false},
			err:   ErrURLMissingRepoName,
		},
		{
			name:  "empty parts 1",
			url:   "https://github.com/luxas/foobar//",
			isOrg: []bool{true, false},
			err:   ErrURLInvalid,
		},
		{
			name:  "empty parts 2",
			url:   "https://github.com//luxas/foobar/",
			isOrg: []bool{true, false},
			err:   ErrURLInvalid,
		},
		{
			name:  "empty URL",
			url:   "",
			isOrg: []bool{true, false},
			err:   ErrURLInvalid,
		},
		{
			name:  "disallow fragments",
			url:   "https://github.com/luxas/foobar#random",
			isOrg: []bool{true, false},
			err:   ErrURLUnsupportedParts,
		},
		{
			name:  "disallow query values",
			url:   "https://github.com/luxas/foobar?foo=bar",
			isOrg: []bool{true, false},
			err:   ErrURLUnsupportedParts,
		},
		{
			name:  "disallow user auth",
			url:   "https://user:pass@github.com/luxas/foobar",
			isOrg: []bool{true, false},
			err:   ErrURLUnsupportedParts,
		},
		{
			name:  "disallow http",
			url:   "http://github.com/luxas/foobar",
			isOrg: []bool{true, false},
			err:   ErrURLUnsupportedScheme,
		},
		{
			name:  "no scheme",
			url:   "github.com/luxas/foobar",
			isOrg: []bool{true, false},
			err:   ErrURLUnsupportedScheme,
		},
		{
			name:  "invalid URL",
			url:   ":foo/bar",
			isOrg: []bool{true, false},
			err:   &url.Error{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.isOrg) == 0 {
				t.Fatal("must set tt.isOrg to one or more values")
			}
			for _, orgParam := range tt.isOrg {
				got, err := ParseRepositoryURL(tt.url, orgParam)
				// Validate so that the error is expected
				validation.TestExpectErrors(t, "ParseRepositoryURL", err, tt.err)
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
					// expect any .git suffix to be removed
					expectedURL = strings.TrimSuffix(expectedURL, ".git")
					if got.String() != expectedURL {
						t.Errorf("ParseRepositoryURL(): got.String() = %q, want %q", got.String(), expectedURL)
					}
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
			name:      "org: https",
			repoinfo:  newOrgRepoInfo("github.com", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeHTTPS,
			want:      "https://github.com/luxas/test-org/other/foo-bar.git",
		},
		{
			name:      "org: git",
			repoinfo:  newOrgRepoInfo("gitlab.com", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeGit,
			want:      "git@gitlab.com:luxas/test-org/other/foo-bar.git",
		},
		{
			name:      "org: ssh",
			repoinfo:  newOrgRepoInfo("my-gitlab.com:6443", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportTypeSSH,
			want:      "ssh://git@my-gitlab.com:6443/luxas/test-org/other/foo-bar",
		},
		{
			name:      "org: none",
			repoinfo:  newOrgRepoInfo("my-gitlab.com:6443", "luxas", []string{"test-org", "other"}, "foo-bar"),
			transport: TransportType("random"),
			want:      "",
		},
		{
			name:      "user: https",
			repoinfo:  newUserRepoInfo("github.com", "luxas", "foo-bar"),
			transport: TransportTypeHTTPS,
			want:      "https://github.com/luxas/foo-bar.git",
		},
		{
			name:      "user: git",
			repoinfo:  newUserRepoInfo("gitlab.com", "luxas", "foo-bar"),
			transport: TransportTypeGit,
			want:      "git@gitlab.com:luxas/foo-bar.git",
		},
		{
			name:      "user: ssh",
			repoinfo:  newUserRepoInfo("my-gitlab.com:6443", "luxas", "foo-bar"),
			transport: TransportTypeSSH,
			want:      "ssh://git@my-gitlab.com:6443/luxas/foo-bar",
		},
		{
			name:      "user: none",
			repoinfo:  newUserRepoInfo("my-gitlab.com:6443", "luxas", "foo-bar"),
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

func TestOrganizationInfo_RefIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		org  OrganizationInfo
		want bool
	}{
		{
			name: "no fields set",
			want: true,
		},
		{
			name: "domain set",
			org: OrganizationInfo{
				Domain: "foo",
			},
			want: false,
		},
		{
			name: "org set",
			org: OrganizationInfo{
				Organization: "bar",
			},
			want: false,
		},
		{
			name: "sub-org set",
			org: OrganizationInfo{
				SubOrganizations: []string{"baz"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.org.RefIsEmpty(); got != tt.want {
				t.Errorf("OrganizationInfo.RefIsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserInfo_RefIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		user UserInfo
		want bool
	}{
		{
			name: "no fields set",
			want: true,
		},
		{
			name: "domain set",
			user: UserInfo{
				Domain: "foo",
			},
			want: false,
		},
		{
			name: "org set",
			user: UserInfo{
				UserLogin: "bar",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.RefIsEmpty(); got != tt.want {
				t.Errorf("UserInfo.RefIsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepositoryInfo_RefIsEmpty(t *testing.T) {
	type fields struct {
		IdentityRef    IdentityRef
		RepositoryName string
	}
	tests := []struct {
		name string
		repo RepositoryInfo
		want bool
	}{
		{
			name: "no fields set",
			want: true,
		},
		{
			name: "user domain set",
			repo: RepositoryInfo{
				IdentityRef: UserInfo{
					Domain: "foo",
				},
			},
			want: false,
		},
		{
			name: "user login set",
			repo: RepositoryInfo{
				IdentityRef: UserInfo{
					UserLogin: "foo",
				},
			},
			want: false,
		},
		{
			name: "org domain set",
			repo: RepositoryInfo{
				IdentityRef: OrganizationInfo{
					Domain: "bar",
				},
			},
			want: false,
		},
		{
			name: "org set",
			repo: RepositoryInfo{
				IdentityRef: OrganizationInfo{
					Organization: "bar",
				},
			},
			want: false,
		},
		{
			name: "sub-org set",
			repo: RepositoryInfo{
				IdentityRef: OrganizationInfo{
					SubOrganizations: []string{"baz"},
				},
			},
			want: false,
		},
		{
			name: "repo set",
			repo: RepositoryInfo{
				RepositoryName: "bar",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.repo.RefIsEmpty(); got != tt.want {
				t.Errorf("RepositoryInfo.RefIsEmpty() = %v, want %v", got, tt.want)
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
			ref:  newUserInfo("github.com", "bar"),
			want: IdentityTypeUser,
		},
		{
			name: "sample top-level org",
			ref:  newOrgInfo("github.com", "bar", nil),
			want: IdentityTypeOrganization,
		},
		{
			name: "sample sub-org",
			ref:  newOrgInfo("github.com", "bar", []string{"baz"}),
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
