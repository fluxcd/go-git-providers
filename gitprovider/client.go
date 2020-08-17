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

// Client is an interface that allows talking to a Git provider.
type Client interface {
	// The Client allows accessing all known resources.
	ResourceClient

	// SupportedDomain returns the domain endpoint for this client, e.g. "github.com", "gitlab.com" or
	// "my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
	// what endpoints.
	// This field is set at client creation time, and can't be changed.
	SupportedDomain() string

	// ProviderID returns the provider ID (e.g. "github", "gitlab") for this client.
	// This field is set at client creation time, and can't be changed.
	ProviderID() ProviderID

	// Raw returns the Go client used under the hood to access the Git provider.
	Raw() interface{}
}

// ResourceClient allows access to resource-specific sub-clients.
type ResourceClient interface {
	// Organizations returns the OrganizationsClient handling sets of organizations.
	Organizations() OrganizationsClient

	// OrgRepositories returns the OrgRepositoriesClient handling sets of repositories in an organization.
	OrgRepositories() OrgRepositoriesClient

	// UserRepositories returns the UserRepositoriesClient handling sets of repositories for a user.
	UserRepositories() UserRepositoriesClient
}

//
//	Clients accessed through the top-level client, returning resource objects.
//

// OrganizationsClient operates on organizations the user has access to.
type OrganizationsClient interface {
	// Get a specific organization the user has access to.
	// This might also refer to a sub-organization.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, o OrganizationRef) (Organization, error)

	// List all top-level organizations the specific user has access to.
	//
	// List returns all available organizations, using multiple paginated requests if needed.
	List(ctx context.Context) ([]Organization, error)

	// Children returns the immediate child-organizations for the specific OrganizationRef o.
	// The OrganizationRef may point to any existing sub-organization.
	//
	// This is not supported in GitHub.
	//
	// Children returns all available organizations, using multiple paginated requests if needed.
	Children(ctx context.Context, o OrganizationRef) ([]Organization, error)

	// Possibly add Create/Update/Delete methods later
}

// OrgRepositoriesClient operates on repositories for organizations.
type OrgRepositoriesClient interface {
	// Get returns the repository for the given reference.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, r OrgRepositoryRef) (OrgRepository, error)

	// List all repositories in the given organization.
	//
	// List returns all available repositories, using multiple paginated requests if needed.
	List(ctx context.Context, o OrganizationRef) ([]OrgRepository, error)

	// Create creates a repository for the given organization, with the data and options.
	//
	// ErrAlreadyExists will be returned if the resource already exists.
	Create(ctx context.Context, r OrgRepositoryRef, req RepositoryInfo, opts ...RepositoryCreateOption) (OrgRepository, error)

	// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	Reconcile(ctx context.Context, r OrgRepositoryRef, req RepositoryInfo, opts ...RepositoryReconcileOption) (resp OrgRepository, actionTaken bool, err error)
}

// UserRepositoriesClient operates on repositories for users.
type UserRepositoriesClient interface {
	// Get returns the repository at the given path.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, r UserRepositoryRef) (UserRepository, error)

	// List all repositories for the given user.
	//
	// List returns all available repositories, using multiple paginated requests if needed.
	List(ctx context.Context, o UserRef) ([]UserRepository, error)

	// Create creates a repository for the given user, with the data and options
	//
	// ErrAlreadyExists will be returned if the resource already exists.
	Create(ctx context.Context, r UserRepositoryRef, req RepositoryInfo, opts ...RepositoryCreateOption) (UserRepository, error)

	// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	Reconcile(ctx context.Context, r UserRepositoryRef, req RepositoryInfo, opts ...RepositoryReconcileOption) (resp UserRepository, actionTaken bool, err error)
}

//
//	Clients accessed through resource objects.
//

// TeamsClient allows reading teams for a specific organization.
// This client can be accessed through Organization.Teams().
type TeamsClient interface {
	// Get a team within the specific organization.
	//
	// name may include slashes, but must not be an empty string.
	// Teams are sub-groups in GitLab.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, name string) (Team, error)

	// List all teams (recursively, in terms of subgroups) within the specific organization.
	//
	// List returns all available organizations, using multiple paginated requests if needed.
	List(ctx context.Context) ([]Team, error)

	// Possibly add Create/Update/Delete methods later
}

// TeamAccessClient operates on the teams list for a specific repository.
// This client can be accessed through Repository.TeamAccess().
type TeamAccessClient interface {
	// Get a team's permission level of this given repository.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, name string) (TeamAccess, error)

	// List the team access control list for this repository.
	//
	// List returns all available team access lists, using multiple paginated requests if needed.
	List(ctx context.Context) ([]TeamAccess, error)

	// Create adds a given team to the repository's team access control list.
	//
	// ErrAlreadyExists will be returned if the resource already exists.
	Create(ctx context.Context, req TeamAccessInfo) (TeamAccess, error)

	// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	Reconcile(ctx context.Context, req TeamAccessInfo) (resp TeamAccess, actionTaken bool, err error)
}

// DeployKeyClient operates on the access credential list for a specific repository.
// This client can be accessed through Repository.DeployKeys().
type DeployKeyClient interface {
	// Get a DeployKey by its name.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, name string) (DeployKey, error)

	// List all deploy keys for the given repository.
	//
	// List returns all available deploy keys for the given type,
	// using multiple paginated requests if needed.
	List(ctx context.Context) ([]DeployKey, error)

	// Create a deploy key with the given specifications.
	//
	// ErrAlreadyExists will be returned if the resource already exists.
	Create(ctx context.Context, req DeployKeyInfo) (DeployKey, error)

	// Reconcile makes sure the given desired state (req) becomes the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	Reconcile(ctx context.Context, req DeployKeyInfo) (resp DeployKey, actionTaken bool, err error)
}
