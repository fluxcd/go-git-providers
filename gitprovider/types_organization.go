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

// OrganizationInfo represents an (top-level- or sub-) organization.
type OrganizationInfo struct {
	// Name is the human-friendly name of this organization, e.g. "Flux" or "Kubernetes SIGs".
	Name *string `json:"name"`

	// Description returns a description for the organization.
	Description *string `json:"description"`
}

// TeamInfo is a representation for a team of users inside of an organization.
type TeamInfo struct {
	// Name describes the name of the team. The team name may contain slashes.
	Name string `json:"name"`

	// Members points to a set of user names (logins) of the members of this team.
	Members []string `json:"members"`
}
