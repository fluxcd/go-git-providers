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
	"net/url"
	"strings"

	"github.com/fluxcd/go-git-providers/validation"
)

// IdentityType is a typed string for what kind of identity type an IdentityRef is.
type IdentityType string

const (
	// IdentityTypeUser represents an identity for a user account.
	IdentityTypeUser = IdentityType("user")
	// IdentityTypeOrganization represents an identity for an organization.
	IdentityTypeOrganization = IdentityType("organization")
	// IdentityTypeSuborganization represents an identity for a sub-organization.
	IdentityTypeSuborganization = IdentityType("suborganization")
)

// IdentityRef references an organization or user account in a Git provider.
type IdentityRef interface {
	// IdentityRef implements ValidateTarget so it can easily be validated as a field.
	validation.ValidateTarget

	// GetDomain returns the URL-domain for the Git provider backend,
	// e.g. "github.com" or "self-hosted-gitlab.com:6443".
	GetDomain() string

	// GetIdentity returns the user account name or a slash-separated path of the
	// <organization-name>[/<sub-organization-name>...] form. This can be used as
	// an identifier for this specific actor in the system.
	GetIdentity() string

	// GetType returns what type of identity this instance represents. If IdentityTypeUser is returned
	// this IdentityRef can safely be casted to an UserRef. If any of IdentityTypeOrganization or
	// IdentityTypeSuborganization are returned, this IdentityRef can be casted to a OrganizationRef.
	GetType() IdentityType

	// String returns the HTTPS URL, and implements fmt.Stringer.
	String() string
}

// RepositoryRef describes a reference to a repository owned by either a user account or organization.
type RepositoryRef interface {
	// RepositoryRef is a superset of IdentityRef.
	IdentityRef

	// GetRepository returns the repository name for this repo.
	GetRepository() string

	// GetCloneURL gets the clone URL for the specified transport type.
	GetCloneURL(transport TransportType) string
}

// UserRef represents a user account in a Git provider.
type UserRef struct {
	// Domain returns e.g. "github.com", "gitlab.com" or a custom domain like "self-hosted-gitlab.com" (GitLab)
	// The domain _might_ contain port information, in the form of "host:port", if applicable
	// +required
	Domain string `json:"domain"`

	// UserLogin returns the user account login name.
	// +required
	UserLogin string `json:"userLogin"`
}

// UserRef implements IdentityRef.
var _ IdentityRef = UserRef{}

// GetDomain returns the the domain part of the endpoint, can include port information.
func (u UserRef) GetDomain() string {
	return u.Domain
}

// GetIdentity returns the identity of this actor, which in this case is the user login name.
func (u UserRef) GetIdentity() string {
	return u.UserLogin
}

// GetType marks this UserRef as being a IdentityTypeUser.
func (u UserRef) GetType() IdentityType {
	return IdentityTypeUser
}

// String returns the HTTPS URL to access the User.
func (u UserRef) String() string {
	return fmt.Sprintf("https://%s/%s", u.GetDomain(), u.GetIdentity())
}

// ValidateFields validates its own fields for a given validator.
func (u UserRef) ValidateFields(validator validation.Validator) {
	// Require the Domain and Organization to be set
	if len(u.Domain) == 0 {
		validator.Required("Domain")
	}
	if len(u.UserLogin) == 0 {
		validator.Required("UserLogin")
	}
}

// OrganizationRef implements IdentityRef.
var _ IdentityRef = OrganizationRef{}

// OrganizationRef is an implementation of OrganizationRef.
type OrganizationRef struct {
	// Domain returns e.g. "github.com", "gitlab.com" or a custom domain like "self-hosted-gitlab.com" (GitLab)
	// The domain _might_ contain port information, in the form of "host:port", if applicable.
	// +required
	Domain string `json:"domain"`

	// Organization specifies the URL-friendly, lowercase name of the organization or user account name,
	// e.g. "fluxcd" or "kubernetes-sigs".
	// +required
	Organization string `json:"organization"`

	// SubOrganizations point to optional sub-organizations (or sub-groups) of the given top-level organization
	// in the Organization field. E.g. "gitlab.com/fluxcd/engineering/frontend" would yield ["engineering", "frontend"]
	// +optional
	SubOrganizations []string `json:"subOrganizations,omitempty"`
}

// GetDomain returns the the domain part of the endpoint, can include port information.
func (o OrganizationRef) GetDomain() string {
	return o.Domain
}

// GetIdentity returns the identity of this actor, which in this case is the user login name.
func (o OrganizationRef) GetIdentity() string {
	orgParts := append([]string{o.Organization}, o.SubOrganizations...)
	return strings.Join(orgParts, "/")
}

// GetType marks this UserRef as being a IdentityTypeUser.
func (o OrganizationRef) GetType() IdentityType {
	if len(o.SubOrganizations) > 0 {
		return IdentityTypeSuborganization
	}
	return IdentityTypeOrganization
}

// String returns the HTTPS URL to access the Organization.
func (o OrganizationRef) String() string {
	return fmt.Sprintf("https://%s/%s", o.GetDomain(), o.GetIdentity())
}

// ValidateFields validates its own fields for a given validator.
func (o OrganizationRef) ValidateFields(validator validation.Validator) {
	// Require the Domain and Organization to be set
	if len(o.Domain) == 0 {
		validator.Required("Domain")
	}
	if len(o.Organization) == 0 {
		validator.Required("Organization")
	}
}

// OrgRepositoryRef is a struct with information about a specific repository owned by an organization.
type OrgRepositoryRef struct {
	// OrgRepositoryRef embeds OrganizationRef inline.
	OrganizationRef `json:",inline"`

	// RepositoryName specifies the Git repository name. This field is URL-friendly,
	// e.g. "kubernetes" or "cluster-api-provider-aws".
	// +required
	RepositoryName string `json:"repositoryName"`
}

// String returns the HTTPS URL to access the repository.
func (r OrgRepositoryRef) String() string {
	return fmt.Sprintf("%s/%s", r.OrganizationRef.String(), r.RepositoryName)
}

// GetRepository returns the repository name for this repo.
func (r OrgRepositoryRef) GetRepository() string {
	return r.RepositoryName
}

// ValidateFields validates its own fields for a given validator.
func (r OrgRepositoryRef) ValidateFields(validator validation.Validator) {
	// First, validate the embedded OrganizationRef
	r.OrganizationRef.ValidateFields(validator)
	// Require RepositoryName to be set
	if len(r.RepositoryName) == 0 {
		validator.Required("RepositoryName")
	}
}

// GetCloneURL gets the clone URL for the specified transport type.
func (r OrgRepositoryRef) GetCloneURL(transport TransportType) string {
	return GetCloneURL(r, transport)
}

// UserRepositoryRef is a struct with information about a specific repository owned by a user.
type UserRepositoryRef struct {
	// UserRepositoryRef embeds UserRef inline.
	UserRef `json:",inline"`

	// RepositoryName specifies the Git repository name. This field is URL-friendly,
	// e.g. "kubernetes" or "cluster-api-provider-aws".
	// +required
	RepositoryName string `json:"repositoryName"`
}

// String returns the HTTPS URL to access the repository.
func (r UserRepositoryRef) String() string {
	return fmt.Sprintf("%s/%s", r.UserRef.String(), r.RepositoryName)
}

// GetRepository returns the repository name for this repo.
func (r UserRepositoryRef) GetRepository() string {
	return r.RepositoryName
}

// ValidateFields validates its own fields for a given validator.
func (r UserRepositoryRef) ValidateFields(validator validation.Validator) {
	// First, validate the embedded OrganizationRef
	r.UserRef.ValidateFields(validator)
	// Require RepositoryName to be set
	if len(r.RepositoryName) == 0 {
		validator.Required("RepositoryName")
	}
}

// GetCloneURL gets the clone URL for the specified transport type.
func (r UserRepositoryRef) GetCloneURL(transport TransportType) string {
	return GetCloneURL(r, transport)
}

// GetCloneURL returns the URL to clone a repository for a given transport type. If the given
// TransportType isn't known an empty string is returned.
func GetCloneURL(rs RepositoryRef, transport TransportType) string {
	switch transport {
	case TransportTypeHTTPS:
		return fmt.Sprintf("%s.git", rs.String())
	case TransportTypeGit:
		return fmt.Sprintf("git@%s:%s/%s.git", rs.GetDomain(), rs.GetIdentity(), rs.GetRepository())
	case TransportTypeSSH:
		return fmt.Sprintf("ssh://git@%s/%s/%s", rs.GetDomain(), rs.GetIdentity(), rs.GetRepository())
	}
	return ""
}

// ParseOrganizationURL parses an URL to an organization into a OrganizationRef object.
func ParseOrganizationURL(o string) (*OrganizationRef, error) {
	u, parts, err := parseURL(o)
	if err != nil {
		return nil, err
	}
	// Create the IdentityInfo object
	info := &OrganizationRef{
		Domain:           u.Host,
		Organization:     parts[0],
		SubOrganizations: []string{},
	}
	// If we've got more than one part, assume they are sub-organizations
	if len(parts) > 1 {
		info.SubOrganizations = parts[1:]
	}
	return info, nil
}

// ParseUserURL parses an URL to an organization into a UserRef object.
func ParseUserURL(u string) (*UserRef, error) {
	// Use the same logic as for parsing organization URLs, but return an UserRef object
	orgInfoPtr, err := ParseOrganizationURL(u)
	if err != nil {
		return nil, err
	}
	userRef, err := orgInfoPtrToUserRef(orgInfoPtr)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, u)
	}
	return userRef, nil
}

// ParseUserRepositoryURL parses a HTTPS clone URL into a UserRepositoryRef object.
func ParseUserRepositoryURL(r string) (*UserRepositoryRef, error) {
	orgInfoPtr, repoName, err := parseRepositoryURL(r)
	if err != nil {
		return nil, err
	}

	userRef, err := orgInfoPtrToUserRef(orgInfoPtr)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrURLInvalid, r)
	}

	return &UserRepositoryRef{
		UserRef:        *userRef,
		RepositoryName: repoName,
	}, nil
}

// ParseOrgRepositoryURL parses a HTTPS clone URL into a OrgRepositoryRef object.
func ParseOrgRepositoryURL(r string) (*OrgRepositoryRef, error) {
	orgInfoPtr, repoName, err := parseRepositoryURL(r)
	if err != nil {
		return nil, err
	}

	return &OrgRepositoryRef{
		OrganizationRef: *orgInfoPtr,
		RepositoryName:  repoName,
	}, nil
}

func parseRepositoryURL(r string) (orgInfoPtr *OrganizationRef, repoName string, err error) {
	// First, parse the URL as an organization
	orgInfoPtr, err = ParseOrganizationURL(r)
	if err != nil {
		return nil, "", err
	}
	// The "repository" part of the URL parsed as an organization, is the last "sub-organization"
	// Check that there's at least one sub-organization
	if len(orgInfoPtr.SubOrganizations) < 1 {
		return nil, "", fmt.Errorf("%w: %s", ErrURLMissingRepoName, r)
	}

	// The repository name is the last "sub-org"
	repoName = orgInfoPtr.SubOrganizations[len(orgInfoPtr.SubOrganizations)-1]
	// Never include any .git suffix at the end of the repository name
	repoName = strings.TrimSuffix(repoName, ".git")

	// Remove the repository name from the sub-org list
	orgInfoPtr.SubOrganizations = orgInfoPtr.SubOrganizations[:len(orgInfoPtr.SubOrganizations)-1]
	return
}

func parseURL(str string) (*url.URL, []string, error) {
	// Fail-fast if the URL is empty
	if len(str) == 0 {
		return nil, nil, fmt.Errorf("url cannot be empty: %w", ErrURLInvalid)
	}
	u, err := url.Parse(str)
	if err != nil {
		return nil, nil, err
	}
	// Only allow explicit https URLs
	if u.Scheme != "https" {
		return nil, nil, fmt.Errorf("%w: %s", ErrURLUnsupportedScheme, str)
	}
	// Don't allow any extra things in the URL, in order to be able to do a successful
	// round-trip of parsing the URL and encoding it back to a string
	if len(u.Fragment) != 0 || len(u.RawQuery) != 0 || len(u.User.String()) != 0 {
		return nil, nil, fmt.Errorf("%w: %s", ErrURLUnsupportedParts, str)
	}

	// Strip any leading and trailing slash to be able to split the string cleanly
	path := strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), "/")
	// Split the path by slash
	parts := strings.Split(path, "/")
	// Make sure there aren't any "empty" string splits
	// This has the consequence that it's guaranteed that there is at least one
	// part returned, so there's no need to check for len(parts) < 1
	for _, p := range parts {
		// Make sure any path part is not empty
		if len(p) == 0 {
			return nil, nil, fmt.Errorf("%w: %s", ErrURLInvalid, str)
		}
	}
	return u, parts, nil
}

func orgInfoPtrToUserRef(orgInfoPtr *OrganizationRef) (*UserRef, error) {
	// Don't tolerate that there are "sub-parts" for an user URL
	if len(orgInfoPtr.SubOrganizations) > 0 {
		return nil, ErrURLInvalid
	}
	// Return an UserRef struct
	return &UserRef{
		Domain:    orgInfoPtr.Domain,
		UserLogin: orgInfoPtr.Organization,
	}, nil
}
