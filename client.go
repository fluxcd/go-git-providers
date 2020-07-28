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

// Client is an interface that allows talking to a Git provider
type Client interface {
	// The Client allows accessing all known resources
	ResourceClient

	// SupportedDomain returns the domain endpoint for this client, e.g. "github.com", "gitlab.com" or
	// "my-custom-git-server.com:6443". This allows a higher-level user to know what Client to use for
	// what endpoints.
	// This field is set at client creation time, and can't be changed
	SupportedDomain() string

	// ProviderID returns the provider ID (e.g. "github", "gitlab") for this client
	// This field is set at client creation time, and can't be changed
	ProviderID() ProviderID

	// Raw returns the Go client used under the hood for accessing the Git provider
	Raw() interface{}
}

// ResourceClient allows access to resource-specific clients
type ResourceClient interface {
	// Organization gets the OrganizationClient for the specific top-level organization, or user account.
	// ErrNotTopLevelOrganization will be returned if the organization is not top-level
	Organization(o OrganizationRef) OrganizationClient

	// Organizations returns the OrganizationsClient handling sets of organizations
	Organizations() OrganizationsClient

	// Repository gets the RepositoryClient for the specified RepositoryRef
	Repository(r RepositoryRef) RepositoryClient

	// Repositories returns the RepositoriesClient handling sets of organizations
	Repositories() RepositoriesClient
}

// OrganizationsClient operates on organizations the user has access to
type OrganizationsClient interface {
	// Get a specific organization the user has access to
	// This might also refer to a sub-organization
	// ErrNotFound is returned if the resource does not exist
	Get(ctx context.Context, o OrganizationRef) (*Organization, error)

	// List all top-level organizations the specific user has access to
	// List should return all available organizations, using multiple paginated requests if needed
	List(ctx context.Context) ([]Organization, error)

	// Children returns the immediate child-organizations for the specific IdentityRef o.
	// The IdentityRef may point to any sub-organization that exists
	// This is not supported in GitHub
	// Children should return all available organizations, using multiple paginated requests if needed
	Children(ctx context.Context, o OrganizationRef) ([]Organization, error)

	// Possibly add Create/Update/Delete methods later
}

// OrganizationClient operates on a given/specific organization
type OrganizationClient interface {
	// Teams gives access to the TeamsClient for this specific organization
	Teams() OrganizationTeamsClient
}

// OrganizationTeamsClient handles teams organization-wide
type OrganizationTeamsClient interface {
	// Get a team within the specific organization
	// teamName may include slashes, to point to e.g. "sub-teams" i.e. subgroups in Gitlab
	// teamName must not be an empty string
	// ErrNotFound is returned if the resource does not exist
	Get(ctx context.Context, teamName string) (*Team, error)

	// List all teams (recursively, in terms of subgroups) within the specific organization
	// List should return all available organizations, using multiple paginated requests if needed
	List(ctx context.Context) ([]Team, error)

	// Possibly add Create/Update/Delete methods later
}

// RepositoriesClient operates on repositories the user has access to
type RepositoriesClient interface {
	// Get returns the repository at the given path
	// ErrNotFound is returned if the resource does not exist
	Get(ctx context.Context, r RepositoryRef) (*Repository, error)

	// List all repositories in the given organization or user account
	// List should return all available repositories, using multiple paginated requests if needed
	List(ctx context.Context, o IdentityRef) ([]Repository, error)

	// Create creates a repository at the given organization path, with the given URL-encoded name and options
	// ErrAlreadyExists will be returned if the resource already exists
	Create(ctx context.Context, r *Repository, opts ...RepositoryCreateOption) (*Repository, error)

	// Update will update the desired state of the repository. Only set fields will be respected.
	// ErrNotFound is returned if the resource does not exist
	Update(ctx context.Context, r *Repository) (*Repository, error)

	// Reconcile makes sure r is the actual state in the backing Git provider. If r doesn't exist
	// under the hood, it is created. If r is already the actual state, this is a no-op. If r isn't
	// the actual state, the resource will be updated.
	Reconcile(ctx context.Context, r *Repository, opts ...RepositoryReconcileOption) (*Repository, error)
}

// RepositoryClient operates on a given/specific repository
type RepositoryClient interface {
	// TeamAccess returns a client for operating on the teams that have access to this specific repository
	TeamAccess() RepositoryTeamAccessClient

	// Credentials gives access to manipulating credentials for accessing this specific repository
	Credentials() RepositoryCredentialsClient
}

// RepositoryTeamAccessClient operates on the teams list for a specific repository
type RepositoryTeamAccessClient interface {
	// Create adds a given team to the repo's team access control list
	// ErrAlreadyExists will be returned if the resource already exists
	// The embedded RepositoryInfo of ta does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Create(ctx context.Context, ta *TeamAccess) error

	// Lists the team access control list for this repo
	List(ctx context.Context) ([]TeamAccess, error)

	// Reconcile makes sure ta is the actual state in the backing Git provider. If ta doesn't exist
	// under the hood, it is created. If ta is already the actual state, this is a no-op. If ta isn't
	// the actual state, the resource will be deleted and recreated.
	// The embedded RepositoryInfo of ta does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Reconcile(ctx context.Context, ta *TeamAccess) (*TeamAccess, error)

	// Delete removes the given team from the repo's team access control list
	// ErrNotFound is returned if the resource does not exist
	Delete(ctx context.Context, ta *TeamAccess) error
}

// RepositoryCredentialsClient operates on the access credential list for a specific repository
type RepositoryCredentialsClient interface {
	// Create a credential with the given human-readable name, the given bytes and optional options
	// ErrAlreadyExists will be returned if the resource already exists
	Create(ctx context.Context, c RepositoryCredential) (RepositoryCredential, error)

	// Lists all credentials for the given credential type
	List(ctx context.Context, t RepositoryCredentialType) ([]RepositoryCredential, error)

	// Reconcile makes sure c is the actual state in the backing Git provider. If c doesn't exist
	// under the hood, it is created. If c is already the actual state, this is a no-op. If c isn't
	// the actual state, the resource will deleted and recreated.
	Reconcile(ctx context.Context, c RepositoryCredential) (RepositoryCredential, error)

	// Deletes a credential from the repo. name corresponds to GetName() of the credential
	// ErrNotFound is returned if the resource does not exist
	Delete(ctx context.Context, c RepositoryCredential) error
}
