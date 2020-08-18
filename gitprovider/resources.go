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

// Organization represents an organization in a Git provider.
// For now, the organization is read-only, i.e. there aren't set/update methods.
type Organization interface {
	// Organization implements the Object interface,
	// allowing access to the underlying object returned from the API.
	Object
	// OrganizationBound returns organization reference details.
	OrganizationBound

	// Get returns high-level information about the organization.
	Get() OrganizationInfo

	// Teams gives access to the TeamsClient for this specific organization
	Teams() TeamsClient
}

// Team represents a team in an organization in a Git provider.
// For now, the team is read-only, i.e. there aren't set/update methods.
type Team interface {
	// Team implements the Object interface,
	// allowing access to the underlying object returned from the API.
	Object
	// OrganizationBound returns organization reference details.
	OrganizationBound

	// Get returns high-level information about this team.
	Get() TeamInfo
}

// UserRepository describes a repository owned by an user.
type UserRepository interface {
	// UserRepository and OrgRepository implement the Object interface,
	// allowing access to the underlying object returned from the API.
	Object
	// The repository can be updated.
	Updatable
	// The repository can be reconciled.
	Reconcilable
	// The repository can be deleted.
	Deletable
	// RepositoryBound returns repository reference details.
	RepositoryBound

	// Get returns high-level information about this repository.
	Get() RepositoryInfo
	// Set sets high-level desired state for this repository. In order to apply these changes in
	// the Git provider, run .Update() or .Reconcile().
	Set(RepositoryInfo) error

	// DeployKeys gives access to manipulating deploy keys to access this specific repository.
	DeployKeys() DeployKeyClient
}

// OrgRepository describes a repository owned by an organization.
type OrgRepository interface {
	// OrgRepository is a superset of UserRepository.
	UserRepository

	// TeamAccess returns a TeamsAccessClient for operating on teams' access to this specific repository.
	TeamAccess() TeamAccessClient
}

// DeployKey represents a short-lived credential (e.g. an SSH public key) used to access a repository.
type DeployKey interface {
	// DeployKey implements the Object interface,
	// allowing access to the underlying object returned from the API.
	Object
	// The deploy key can be updated.
	Updatable
	// The deploy key can be reconciled.
	Reconcilable
	// The deploy key can be deleted.
	Deletable
	// RepositoryBound returns repository reference details.
	RepositoryBound

	// Get returns high-level information about this deploy key.
	Get() DeployKeyInfo
	// Set sets high-level desired state for this deploy key. In order to apply these changes in
	// the Git provider, run .Update() or .Reconcile().
	Set(DeployKeyInfo) error
}

// TeamAccess describes a binding between a repository and a team.
type TeamAccess interface {
	// TeamAccess implements the Object interface,
	// allowing access to the underlying object returned from the API.
	Object
	// The deploy key can be updated.
	Updatable
	// The deploy key can be reconciled.
	Reconcilable
	// The deploy key can be deleted.
	Deletable
	// RepositoryBound returns repository reference details.
	RepositoryBound

	// Get returns high-level information about this team access for the repository.
	Get() TeamAccessInfo
	// Set sets high-level desired state for this team access object. In order to apply these changes in
	// the Git provider, run .Update() or .Reconcile().
	Set(TeamAccessInfo) error
}
