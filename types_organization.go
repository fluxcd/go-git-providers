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

// Organization implements the Object and OrganizationRef interfaces
var _ Object = &Organization{}
var _ OrganizationRef = &Organization{}

// Organization represents an (top-level- or sub-) organization
type Organization struct {
	// OrganizationInfo provides the required fields
	// (Domain, Organization and SubOrganizations) required for being an OrganizationRef
	OrganizationInfo `json:",inline"`
	// InternalHolder implements the InternalGetter interface
	// +optional
	InternalHolder `json:",inline"`

	// Name is the human-friendly name of this organization, e.g. "Weaveworks" or "Kubernetes SIGs"
	// +required
	Name string `json:"name"`

	// Description returns a description for the organization
	// No default value at POST-time
	// +optional
	Description *string `json:"description"`
}

// Team implements the Object interface
var _ Object = &Team{}

// Team is a representation for a team of users inside of an organization
type Team struct {
	// Team embeds InternalHolder for accessing the underlying object
	// +optional
	InternalHolder `json:",inline"`

	// Name describes the name of the team. The team name may contain slashes
	// +required
	Name string `json:"name"`

	// Members points to a set of user names (logins) of the members of this team
	// +required
	Members []string `json:"members"`

	// Organization specifies the information about what organization this Team is associated with.
	// It is populated in .Get() and .List() calls.
	// When creating, this field is optional. However, if specified, it must match the OrganizationRef
	// given to the client.
	// +optional
	Organization *OrganizationInfo `json:"organization"`
}
