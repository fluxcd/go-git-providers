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

const (
	// the default repository visibility is private
	defaultRepoVisibility = RepoVisibilityPrivate
	// the default repository permission is "pull" (or read)
	defaultRepoPermission = RepositoryPermissionPull
)

// Repository implements Object and RepositoryRef interfaces
// Repository can be created and updated
var _ Object = &Repository{}
var _ RepositoryRef = &Repository{}
var _ Creator = &Repository{}
var _ Updator = &Repository{}

// Repository represents a Git repository provided by a Git provider
type Repository struct {
	// RepositoryInfo provides the required fields
	// (Domain, Organization, SubOrganizations and RepositoryName)
	// required for being an RepositoryRef
	RepositoryInfo `json:",inline"`
	// InternalHolder implements the InternalGetter interface
	// +optional
	InternalHolder `json:",inline"`

	// Description returns a description for the repository
	// No default value at POST-time
	// +optional
	Description *string `json:"description"`

	// Visibility returns the desired visibility for the repository
	// Default value at POST-time: RepoVisibilityPrivate
	// +optional
	Visibility *RepoVisibility
}

// Default defaults the Repository, implementing the Creator interface
func (r *Repository) Default() {
	if r.Visibility == nil {
		visibility := defaultRepoVisibility
		r.Visibility = &visibility
	}
}

// ValidateCreate validates the object at POST-time and implements the Creator interface
func (r *Repository) ValidateCreate() error {
	errs := newValidationErrorList("Repository")
	// Validate the embedded RepositoryInfo (and its OrganizationInfo)
	r.RepositoryInfo.validateRepositoryInfoCreate(errs)
	return errs.Error()
}

// ValidateUpdate validates the object at PUT/PATCH-time and implements the Updator interface
func (r *Repository) ValidateUpdate() error {
	// No specific update logic, just make sure required fields are set
	return r.ValidateCreate()
}

// TeamAccess implements Object and RepositoryRef interfaces
// TeamAccess can be created and deleted
var _ Object = &TeamAccess{}
var _ Creator = &TeamAccess{}
var _ Deletor = &TeamAccess{}

// TeamAccess describes a binding between a repository and a team
type TeamAccess struct {
	// TeamAccess embeds InternalHolder for accessing the underlying object
	// +optional
	InternalHolder `json:",inline"`

	// Name describes the name of the team. The team name may contain slashes
	// +required
	Name string `json:"name"`

	// Permission describes the permission level for which the team is allowed to operate
	// Default: pull
	// Available options: See the RepositoryPermission enum
	// +optional
	Permission *RepositoryPermission

	// Repository specifies the information about what repository this TeamAccess is associated with.
	// It is populated in .Get() and .List() calls.
	// When creating, this field is optional. However, if specified, it must match the RepositoryRef
	// given to the client.
	// +optional
	Repository RepositoryInfo `json:"repository"`
}

// Default defaults the TeamAccess, implementing the Creator interface
func (ta *TeamAccess) Default() {
	if ta.Permission == nil {
		permission := defaultRepoPermission
		ta.Permission = &permission
	}
}

// ValidateCreate validates the object at POST-time and implements the Creator interface
func (ta *TeamAccess) ValidateCreate() error {
	errs := newValidationErrorList("TeamAccess")
	// Make sure we've set the name of the team
	if len(ta.Name) == 0 {
		errs.Required("Name")
	}
	// Don't care about the RepositoryInfo, as that information is coming from
	// the RepositoryClient. In the client, we make sure that they equal.
	return errs.Error()
}

// ValidateDelete validates the object at DELETE-time and implements the Deletor interface
func (ta *TeamAccess) ValidateDelete() error {
	// No specific update logic, just make sure required fields are set
	return ta.ValidateCreate()
}
