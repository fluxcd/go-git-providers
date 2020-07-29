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
	//
	// ErrNotTopLevelOrganization will be returned at usage time if the organization is not top-level.
	Organization(o OrganizationRef) OrganizationClient

	// Organizations returns the OrganizationsClient handling sets of organizations.
	Organizations() OrganizationsClient

	// Repository gets the RepositoryClient for the specified RepositoryRef.
	Repository(r RepositoryRef) RepositoryClient

	// Repositories returns the RepositoriesClient handling sets of organizations.
	Repositories() RepositoriesClient
}

// OrganizationsClient operates on organizations the user has access to.
type OrganizationsClient interface {
	// Get a specific organization the user has access to.
	// This might also refer to a sub-organization.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, o OrganizationRef) (*Organization, error)

	// List all top-level organizations the specific user has access to.
	//
	// List should return all available organizations, using multiple paginated requests if needed.
	List(ctx context.Context) ([]Organization, error)

	// Children returns the immediate child-organizations for the specific OrganizationRef o.
	// The OrganizationRef may point to any sub-organization that exists.
	//
	// This is not supported in GitHub.
	//
	// Children should return all available organizations, using multiple paginated requests if needed.
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
	// Get a team within the specific organization.
	//
	// teamName may include slashes, to point to e.g. subgroups in Gitlab.
	// teamName must not be an empty string.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, teamName string) (*Team, error)

	// List all teams (recursively, in terms of subgroups) within the specific organization
	//
	// List should return all available organizations, using multiple paginated requests if needed.
	List(ctx context.Context) ([]Team, error)

	// Possibly add Create/Update/Delete methods later
}

// RepositoriesClient operates on repositories the user has access to
type RepositoriesClient interface {
	// Get returns the repository at the given path.
	//
	// ErrNotFound is returned if the resource does not exist.
	Get(ctx context.Context, r RepositoryRef) (*Repository, error)

	// List all repositories in the given organization or user account.
	//
	// List returns all available repositories, using multiple paginated requests if needed.
	List(ctx context.Context, o IdentityRef) ([]Repository, error)

	// Create creates a repository at the given organization path, with the given URL-encoded name and options
	//
	// ErrAlreadyExists will be returned if the resource already exists.
	//
	// resp will contain any updated information given by the server; hence it is encouraged
	// to stop using req after this call, and use resp instead.
	Create(ctx context.Context, req *Repository, opts ...RepositoryCreateOption) (resp *Repository, err error)

	// Update will update the desired state of the repository. Only set fields will be respected.
	//
	// ErrNotFound is returned if the resource does not exist.
	//
	// resp will contain any updated information given by the server; hence it is encouraged
	// to stop using req after this call, and use resp instead.
	Update(ctx context.Context, req *Repository) (resp *Repository, err error)

	// Reconcile makes sure req is the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	//
	// resp will contain any updated information given by the server; hence it is encouraged
	// to stop using req after this call, and use resp instead.
	Reconcile(ctx context.Context, req *Repository, opts ...RepositoryReconcileOption) (resp *Repository, actionTaken bool, err error)
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
	// List lists the team access control list for this repository.
	//
	// List returns all available team access lists, using multiple paginated requests if needed.
	List(ctx context.Context) ([]TeamAccess, error)

	// Create adds a given team to the repo's team access control list.
	//
	// ErrAlreadyExists will be returned if the resource already exists.
	//
	// resp will contain any updated information given by the server; hence it is encouraged
	// to stop using req after this call, and use resp instead.
	//
	// req.Repository does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Create(ctx context.Context, req *TeamAccess) (resp *TeamAccess, err error)

	// Delete removes the given team from the repo's team access control list.
	//
	// ErrNotFound is returned if the resource does not exist.
	//
	// req.Repository does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Delete(ctx context.Context, req *TeamAccess) error

	// Reconcile makes sure req is the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	//
	// resp will contain any updated information given by the server; hence it is encouraged
	// to stop using req after this call, and use resp instead.
	//
	// req.Repository does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Reconcile(ctx context.Context, req *TeamAccess) (resp *TeamAccess, actionTaken bool, err error)
}

// RepositoryCredentialsClient operates on the access credential list for a specific repository
type RepositoryCredentialsClient interface {
	// List lists all repository credentials of the given credential type.
	//
	// List returns all available repository credentials for the given type,
	// using multiple paginated requests if needed.
	List(ctx context.Context, t RepositoryCredentialType) ([]RepositoryCredential, error)

	// Create creates a credential with the given specifications.
	//
	// ErrAlreadyExists will be returned if the resource already exists.
	//
	// resp will contain any updated information given by the server; hence it is encouraged
	// to stop using req after this call, and use resp instead.
	//
	// req.GetRepositoryRef() does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Create(ctx context.Context, req RepositoryCredential) (resp RepositoryCredential, err error)

	// Delete deletes a credential from the repository.
	//
	// ErrNotFound is returned if the resource does not exist.
	//
	// req.GetRepositoryRef() does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Delete(ctx context.Context, req RepositoryCredential) error

	// Reconcile makes sure req is the actual state in the backing Git provider.
	//
	// If req doesn't exist under the hood, it is created (actionTaken == true).
	// If req doesn't equal the actual state, the resource will be deleted and recreated (actionTaken == true).
	// If req is already the actual state, this is a no-op (actionTaken == false).
	//
	// resp will contain any updated information given by the server; hence it is encouraged
	// to stop using req after this call, and use resp instead.
	//
	// req.GetRepositoryRef() does not need to be populated, but if it is,
	// it must equal to the RepositoryRef given to the RepositoryClient.
	Reconcile(ctx context.Context, req RepositoryCredential) (resp RepositoryCredential, actionTaken bool, err error)
}
