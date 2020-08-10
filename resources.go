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

import "context"

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
	GenericUpdatable
	GenericDeletable
	// RepositoryBound returns repository reference details.
	RepositoryBound

	// Reconcile makes sure req (== this object) is the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	//
	// The internal API object will be overridden with the received server data if actionTaken == true.
	Reconcile(ctx context.Context, opts ...RepositoryReconcileOption) (actionTaken bool, err error)

	// Get returns high-level information about this repository.
	Get() RepositoryInfo
	// Set sets high-level desired state for this repository. In order to apply these changes in
	// the Git provider, use
	Set(i RepositoryInfo, applyToServer bool) error

	// DeployKeys gives access to manipulating deploy keys for accessing this specific repository
	DeployKeys() DeployKeyClient
}

// OrgRepository describes a respository owned by an organization.
type OrgRepository interface {
	UserRepository

	// TeamAccess returns a client for operating on the teams that have access to this specific repository
	TeamAccess() TeamAccessClient
}

// DeployKey represents a short-lived credential (e.g. an SSH public key) used for accessing a repository
type DeployKey interface {
	// DeployKey implements the Object interface,
	// allowing access to the underlying object returned from the API.
	Object
	GenericReconcilable
	GenericDeletable
	// RepositoryBound returns repository reference details.
	RepositoryBound

	// Get returns high-level information about this deploy key.
	Get() DeployKeyInfo
	Set(DeployKeyInfo) error
}

// TeamAccess describes a binding between a repository and a team
type TeamAccess interface {
	// TeamAccess implements the Object interface,
	// allowing access to the underlying object returned from the API.
	Object
	GenericUpdatable
	GenericReconcilable
	GenericDeletable
	// RepositoryBound returns repository reference details.
	RepositoryBound

	// Get returns high-level information about this team access for the repository.
	Get() TeamAccessInfo
	Set(TeamAccessInfo) error
}
