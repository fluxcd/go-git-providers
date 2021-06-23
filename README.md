# go-git-providers

[![godev](https://img.shields.io/static/v1?label=godev&message=reference&color=00add8)](https://pkg.go.dev/github.com/fluxcd/go-git-providers)
[![build](https://github.com/fluxcd/go-git-providers/workflows/build/badge.svg)](https://github.com/fluxcd/go-git-providers/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/fluxcd/go-git-providers)](https://goreportcard.com/report/github.com/fluxcd/go-git-providers)
[![codecov.io](https://codecov.io/github/fluxcd/go-git-providers/coverage.svg?branch=master)](https://codecov.io/github/fluxcd/go-git-providers?branch=master)
[![LICENSE](https://img.shields.io/github/license/fluxcd/go-git-providers)](https://github.com/fluxcd/go-git-providers/blob/master/LICENSE)
[![Contributors](https://img.shields.io/github/contributors/fluxcd/go-git-providers)](https://github.com/fluxcd/go-git-providers/graphs/contributors)
[![Release](https://img.shields.io/github/v/release/fluxcd/go-git-providers?include_prereleases)](https://github.com/fluxcd/go-git-providers/releases/latest)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](https://github.com/fluxcd/go-git-providers/blob/master/CONTRIBUTING.md)

[go-git-providers](https://pkg.go.dev/github.com/fluxcd/go-git-providers) is a general-purpose Go client for interacting with Git providers' APIs (e.g. GitHub, GitLab, Bitbucket).

## Features

- **Consistency:** Using the same Client interface and high-level structs for multiple backends.
- **Authentication:** Personal Access Tokens/OAuth2 Tokens, and unauthenticated.
- **Pagination:** List calls automatically return all available pages.
- **Conditional Requests:** Asks the Git provider if cached data is up-to-date before requesting, to avoid being rate limited.
- **Reconciling:** Support reconciling desired state towards actual state and drift detection.
- **Low-level access:** Access the underlying, provider-specific data easily, if needed, and support applying it to the server.
- **Wrapped errors:** Data-rich, Go 1.14-errors are consistent across provider, including cases like rate limit, validation, not found, etc.
- **Go modules:** The major version is bumped if breaking changes, or major library upgrades are made.
- **Validation-first:** Both server and user data is validated prior to manipulation.
- **URL Parsing:** HTTPS user, organization and repository URLs can be parsed into machine-readable structs.
- **Enums:** Consistent enums are used across providers for similar lists of values.
- **Domain customization:** The user can specify their desired domain for the Git provider backend.
- **Context-first:** `context.Context` is the first parameter for every API call.

## Operations and Design

The top-level `gitprovider.Client` has the following sub-clients with their described capabilities:

- `OrganizationsClient` operates on organizations the user has access to.
  - `Get` a specific organization the user has access to.
  - `List` all top-level organizations the specific user has access to.
  - `Children` returns the immediate child-organizations for the specific OrganizationRef.

- `{Org,User}RepositoriesClient` operates on repositories for organizations and users, respectively.
  - `Get` returns the repository for the given reference.
  - `List` all repositories in the given organization or user account.
  - `Create` creates a repository, with the specified data and options.
  - `Reconcile` makes sure the given desired state becomes the actual state in the backing Git provider.

The sub-clients above return `gitprovider.Organization` or `gitprovider.{Org,User}Repository` interfaces.
These object interfaces lets you access their data (through their `.Get()` function), internal,
provider-specific representation (through their `.APIObject()` function), or sub-resources like deploy keys
and teams.

The following object-scoped clients are available:

- `Organization` represents an organization in a Git provider.
  - `Teams` gives access to the `TeamsClient` for this specific organization.
    - `Get` a team within the specific organization.
    - `List` all teams within the specific organization.

- `UserRepository` describes a repository owned by an user.
  - `DeployKeys` gives access to manipulating deploy keys, using this `DeployKeyClient`.
    - `Get` a DeployKey by its name.
    - `List` all deploy keys for the given repository.
    - `Create` a deploy key with the given specifications.
    - `Reconcile` makes sure the given desired state becomes the actual state in the backing Git provider.

- `OrgRepository` is a superset of `UserRepository`, and describes a repository owned by an organization.
  - `DeployKeys` as in `UserRepository`.
  - `TeamAccess` returns a `TeamsAccessClient` for operating on teams' access to this specific repository.
    - `Get` a team's permission level of this given repository.
    - `List` the team access control list for this repository.
    - `Create` adds a given team to the repository's team access control list.
    - `Reconcile` makes sure the given desired state (req) becomes the actual state in the backing Git provider.

Wait, how do I `Delete` or `Update` an object?

That's done on the returned objects themselves, using the following `Updatable`, `Reconcilable` and `Deletable`
interfaces implemented by `{Org,User}Repository`, `DeployKey` and `TeamAccess`:

```go
// Updatable is an interface which all objects that can be updated
// using the Client implement.
type Updatable interface {
    // Update will apply the desired state in this object to the server.
    // Only set fields will be respected (i.e. PATCH behaviour).
    // In order to apply changes to this object, use the .Set({Resource}Info) error
    // function, or cast .APIObject() to a pointer to the provider-specific type
    // and set custom fields there.
    //
    // ErrNotFound is returned if the resource does not exist.
    //
    // The internal API object will be overridden with the received server data.
    Update(ctx context.Context) error
}

// Deletable is an interface which all objects that can be deleted
// using the Client implement.
type Deletable interface {
    // Delete deletes the current resource irreversibly.
    //
    // ErrNotFound is returned if the resource doesn't exist anymore.
    Delete(ctx context.Context) error
}

// Reconcilable is an interface which all objects that can be reconciled
// using the Client implement.
type Reconcilable interface {
    // Reconcile makes sure the desired state in this object (called "req" here) becomes
    // the actual state in the backing Git provider.
    //
    // If req doesn't exist under the hood, it is created (actionTaken == true).
    // If req doesn't equal the actual state, the resource will be updated (actionTaken == true).
    // If req is already the actual state, this is a no-op (actionTaken == false).
    //
    // The internal API object will be overridden with the received server data if actionTaken == true.
    Reconcile(ctx context.Context) (actionTaken bool, err error)
}
```

In order to access the provider-specific, internal object, all resources implement the `gitprovider.Object` interface:

```go
// Object is the interface all types should implement.
type Object interface {
    // APIObject returns the underlying value that was returned from the server.
    // This is always a pointer to a struct.
    APIObject() interface{}
}
```

So, how do I set the desired state for an object before running `Update` or `Reconcile`?

Using the `Get() {Resource}Info` or `Set({Resource}Info) error` methods. An example as follows, for `TeamAccess`:

```go
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

// TeamAccessInfo contains high-level information about a team's access to a repository.
type TeamAccessInfo struct {
    // Name describes the name of the team. The team name may contain slashes.
    // +required
    Name string `json:"name"`

    // Permission describes the permission level for which the team is allowed to operate.
    // Default: pull.
    // Available options: See the RepositoryPermission enum.
    // +optional
    Permission *RepositoryPermission `json:"permission,omitempty"`
}
```

## Examples

See the following (automatically tested) examples:

- [github/example_organization_test.go](github/example_organization_test.go)
- [github/example_repository_test.go](github/example_repository_test.go)


If you need to run `make test` for your fork/branch you may need to supply the following environment variables:
- GIT_PROVIDER_ORGANIZATION : For GitHub this should be an organization whereas for GitLab this should be a top-level group. As this environment variable is used for both test suites, the name of the GitHub organization must match the name of the GitLab top-level group. Also, this organization needs to have its repository default branch set to `main`. For stash specify and existing project.
- GIT_PROVIDER_USER : This should be the same username for both GitHub and GitLab.
- GITLAB_TEST_TEAM_NAME : An existing GitLab group.
- GITLAB_TEST_SUBGROUP : An existing GitLab subgroup of the GIT_PROVIDER_ORGANIZATION top-level group.
- GITPROVIDER_BOT_TOKEN : A GitHub token with `repo`, `admin:org` and `delete_repo` permissions.
- GITLAB_TOKEN: A GitLab token with `api` scope.
- TEST_VERBOSE: Set to '-v' to emit test output for debugging purposes
- CLEANUP_ALL: Set to delete all test repos after testing.
- TEST_PATTERN: Use to run only matching testsm i.e. `./stash` for stash provider tests only.
- TEST_STOP_ON_ERROR: Set to `-failfast` to stop on first test failure, currently ineffective due to go test bug.
- HTTP_CLIENT_DEBUG: Set to `true` to emitt http requests and responses, will expose credentials.
- STASH_TEST_REPO_NAME: The name of an existing repository in the `GIT_PROVIDER_ORGANIZATION` project.
- STASH_TEST_TEAM_NAME: An existing group.
- STASH_TOKEN: A Stash token.
- STASH_DOMAIN: Domain name of the stash server.

## Maintainers

In alphabetical order:

- Mike Beaumont, [@michaelbeaumont](https://github.com/michaelbeaumont)
- Sara El-Zayat, [@sarataha](https://github.com/sarataha)
- Simon Howe, [@foot](https://github.com/foot)
- Dinos Kousidis, [@dinosk](https://github.com/dinosk)
- Stefan Prodan, [@stefanprodan](https://github.com/stefanprodan)
- Yiannis Triantafyllopoulos, [@yiannistri](https://github.com/yiannistri)

## Getting Help

If you have any questions about this library:

- Read [the pkg.go.dev reference](https://pkg.go.dev/github.com/fluxcd/go-git-providers).
- Invite yourself to the <a href="https://slack.cncf.io" target="_blank">CNCF community</a>
  slack and ask a question on the [#flux](https://cloud-native.slack.com/messages/flux/)
  channel.
- To be part of the conversation about Flux's development, join the
  [flux-dev mailing list](https://lists.cncf.io/g/cncf-flux-dev).
- [File an issue.](https://github.com/fluxcd/go-git-providers/issues/new)

Your feedback is always welcome!

## License

[Apache 2.0](LICENSE)
