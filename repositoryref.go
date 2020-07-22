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
)

// TODO: Add equality methods for OrganizationRef and RepositoryRefs

// OrganizationRef references an organization in a Git provider
type OrganizationRef interface {
	// String returns the HTTPS URL
	fmt.Stringer

	// GetDomain returns the URL-domain for the Git provider backend, e.g. github.com or self-hosted-gitlab.com
	GetDomain() string
	// GetOrganization returns the top-level organization, i.e. "fluxcd" or "kubernetes-sigs"
	GetOrganization() string
	// GetSubOrganizations returns the names of sub-organizations (or sub-groups),
	// e.g. ["engineering", "frontend"] would be returned for gitlab.com/fluxcd/engineering/frontend
	GetSubOrganizations() []string
	// RefIsEmpty returns true if all the parts of a given Organization or Repository reference are empty, otherwise false
	RefIsEmpty() bool
}

// RepositoryRef references a repository hosted by a Git provider
type RepositoryRef interface {
	// RepositoryRef requires an OrganizationRef to fully-qualify a repo reference
	OrganizationRef

	// GetRepository returns the name of the repository
	GetRepository() string
}

// OrganizationInfo implements OrganizationRef
type OrganizationInfo struct {
	// Domain returns e.g. "github.com", "gitlab.com" or a custom domain like "self-hosted-gitlab.com" (GitLab)
	// The domain _might_ contain port information, in the form of "host:port", if applicable
	// +required
	Domain string `json:"domain"`

	// Organization specifies the URL-friendly, lowercase name of the organization, e.g. "fluxcd" or "kubernetes-sigs".
	// +required
	Organization string `json:"organization"`

	// SubOrganizations point to optional sub-organizations (or sub-groups) of the given top-level organization
	// in the Organization field. E.g. "gitlab.com/fluxcd/engineering/frontend" would yield ["engineering", "frontend"]
	// +optional
	SubOrganizations []string `json:"subOrganizations"`
}

// OrganizationInfo implements OrganizationRef
var _ OrganizationRef = OrganizationInfo{}

// GetDomain returns the the domain
func (o OrganizationInfo) GetDomain() string {
	return o.Domain
}

// GetOrganization returns the top-level organization
func (o OrganizationInfo) GetOrganization() string {
	return o.Organization
}

// GetOrganization returns the top-level organization
func (o OrganizationInfo) GetSubOrganizations() []string {
	return o.SubOrganizations
}

// organizationString returns the organizations of an OrganizationRef slash-separated
func organizationString(o OrganizationRef) string {
	orgParts := append([]string{o.GetOrganization()}, o.GetSubOrganizations()...)
	return strings.Join(orgParts, "/")
}

// String returns the HTTPS URL to access the Organization
func (o OrganizationInfo) String() string {
	return fmt.Sprintf("https://%s/%s", o.GetDomain(), organizationString(o))
}

// RefIsEmpty returns true if all the parts of the given OrganizationInfo are empty, otherwise false
func (o OrganizationInfo) RefIsEmpty() bool {
	return len(o.Domain) == 0 && len(o.Organization) == 0 && len(o.SubOrganizations) == 0
}

// validateOrganizationInfoCreate validates its own field into a given error list
func (o OrganizationInfo) validateOrganizationInfoCreate(errs *validationErrorList) {
	// Require the Domain and Organization to be set
	if len(o.Domain) == 0 {
		errs.Required("Domain")
	}
	if len(o.Organization) == 0 {
		errs.Required("Organization")
	}
}

// RepositoryInfo is an implementation of RepositoryRef
type RepositoryInfo struct {
	// RepositoryInfo embeds everything in OrganizationInfo inline
	OrganizationInfo `json:",inline"`

	// Name specifies the Git repository name. This field is URL-friendly,
	// e.g. "kubernetes" or "cluster-api-provider-aws"
	// +required
	RepositoryName string `json:"repositoryName"`
}

// RepositoryInfo implements the RepositoryRef interface
var _ RepositoryRef = RepositoryInfo{}

// GetRepository returns the name of the repository
func (r RepositoryInfo) GetRepository() string {
	return r.RepositoryName
}

// String returns the HTTPS URL to access the Repository
func (r RepositoryInfo) String() string {
	return fmt.Sprintf("%s/%s", r.OrganizationInfo.String(), r.GetRepository())
}

// RefIsEmpty returns true if all the parts of the given RepositoryInfo are empty, otherwise false
func (r RepositoryInfo) RefIsEmpty() bool {
	return r.OrganizationInfo.RefIsEmpty() && len(r.RepositoryName) == 0
}

// validateRepositoryInfoCreate validates its own field into a given error list
func (r RepositoryInfo) validateRepositoryInfoCreate(errs *validationErrorList) {
	// First, validate the embedded OrganizationInfo
	r.validateOrganizationInfoCreate(errs)
	// Require RepositoryName to be set
	if len(r.RepositoryName) == 0 {
		errs.Required("RepositoryName")
	}
}

// GetCloneURL gets the clone URL for the specified transport type
func (r RepositoryInfo) GetCloneURL(transport TransportType) string {
	return GetCloneURL(r, transport)
}

// GetCloneURL returns the URL to clone a repository for a given transport type. If the given
// TransportType isn't known an empty string is returned.
func GetCloneURL(rs RepositoryRef, transport TransportType) string {
	switch transport {
	case TransportTypeHTTPS:
		return fmt.Sprintf("%s.git", rs.String())
	case TransportTypeGit:
		return fmt.Sprintf("git@%s:%s/%s.git", rs.GetDomain(), organizationString(rs), rs.GetRepository())
	case TransportTypeSSH:
		return fmt.Sprintf("ssh://git@%s/%s/%s", rs.GetDomain(), organizationString(rs), rs.GetRepository())
	}
	return ""
}

// ParseOrganizationURL parses an URL to an organization into a OrganizationRef object
func ParseOrganizationURL(o string) (OrganizationRef, error) {
	// Always return OrganizationInfo dereferenced, not as a pointer
	orgInfoPtr, err := parseOrganizationURL(o)
	if err != nil {
		return nil, err
	}
	return *orgInfoPtr, nil
}

// ParseRepositoryURL parses a HTTPS or SSH clone URL into a RepositoryRef object
func ParseRepositoryURL(r string) (RepositoryRef, error) {
	// First, parse the URL as an organization
	orgInfoPtr, err := parseOrganizationURL(r)
	if err != nil {
		return nil, err
	}
	// The "repository" part of the URL parsed as an organization, is the last "sub-organization"
	// Check that there's at least one sub-organization
	if len(orgInfoPtr.SubOrganizations) < 1 {
		return nil, fmt.Errorf("%w: %s", ErrURLMissingRepoName, r)
	}

	// The repository name is the last "sub-org"
	repoName := orgInfoPtr.SubOrganizations[len(orgInfoPtr.SubOrganizations)-1]
	// Remove the repository name from the sub-org list
	orgInfoPtr.SubOrganizations = orgInfoPtr.SubOrganizations[:len(orgInfoPtr.SubOrganizations)-1]

	// Return the new RepositoryInfo
	return RepositoryInfo{
		RepositoryName:   repoName,
		OrganizationInfo: *orgInfoPtr,
	}, nil
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

func parseOrganizationURL(o string) (*OrganizationInfo, error) {
	u, parts, err := parseURL(o)
	if err != nil {
		return nil, err
	}
	// Create the OrganizationInfo object
	info := &OrganizationInfo{
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
