# Development

> **Note**
> Please take a look at <https://fluxcd.io/contributing/flux/>
> to find out about how to contribute to Flux and how to interact with the
> Flux Development team.

## Requirements

- Go >= 1.18
- Make

## How to run the test suite locally

The test suite can broadly be divided into unit and end-to-end/integration tests. The `test` Make target runs the unit tests:

```
make test
```

For more verbose test output, use `TEST_FLAGS`:

```
make test TEST_FLAGS="-v"
```

### End-to-end tests

> **Note**
> Set the `CLEANUP_ALL` environment variable to a non-empty value if you want the e2e test suite to delete
> any data that is created as part of the test run. Otherwise your test group/organization will be left behind
> with lots of repositories.

The end-to-end test suite has tests for all supported providers, each backed by a dedicated Make target.

#### GitHub

```
make test-e2e-github
```

All tests are run against github.com. Adjust the following variables to your needs:

| Setting           | Default value                 | Environment variable        |
| ----------------- | ----------------------------- | --------------------------- |
| Access token      | read from `/tmp/github-token` | `GITHUB_TOKEN`              |
| Test organization | fluxcd-testing                | `GIT_PROVIDER_ORGANIZATION` |
| Test user         | fluxcd-gitprovider-bot        | `GIT_PROVIDER_USER`         |

#### GitLab

For the GitLab tests there is automation in place to spin up an ephemeral GitLab instance to run the test suite against:

```
make start-provider-instances-gitlab
```

As soon as the containers are up and GitLab is running, execute the tests:

```
make test-e2e-gitlab
```

The Make target automatically runs the tests against the ephemeral instance. To change the test configuration, adjust
the following variables to your needs:

| Setting                                         | Default value                 | Environment variable        |
| ----------------------------------------------- | ----------------------------- | --------------------------- |
| Access token                                    | read from `/tmp/gitlab-token` | `GITLAB_TOKEN`              |
| Test group                                      | fluxcd-testing                | `GIT_PROVIDER_ORGANIZATION` |
| Test subgroup (nested under the test group)     | fluxcd-testing-sub-group      | `GITLAB_TEST_SUBGROUP`      |
| Test team (this is an ordinary group in GitLab) | fluxcd-testing-2              | `GITLAB_TEST_TEAM_NAME`     |
| Test user                                       | fluxcd-gitprovider-bot        | `GIT_PROVIDER_USER`         |

#### Stash

```
make test-e2e-stash
```

| Setting           | Default value                | Environment variable        |
| ----------------- | ---------------------------- | --------------------------- |
| Domain            | stash.example.com            | `STASH_DOMAIN`              |
| Access token      | read from `/tmp/stash.token` | `STASH_TOKEN`               |
| Test user         |                              | `STASH_USER`                |
| Test organization | go-git-provider-testing      | `GIT_PROVIDER_ORGANIZATION` |
| Test team         | fluxcd-test-team             | `STASH_TEST_TEAM_NAME`      |


#### Gitea

For the Gitea tests there is automation in place to spin up an ephemeral Gitea instance to run the test suite against:

```
make start-provider-instances-gitea
```

As soon as the containers are up and Gitea is running, execute the tests:

```
make test-e2e-gitea
```

The Make target automatically runs the tests against the ephemeral instance. To change the test configuration, adjust
the following variables to your needs:

| Setting                                         | Default value                 | Environment variable        |
| ----------------------------------------------- | ----------------------------- | --------------------------- |
| Access token                                    | read from `/tmp/gitea-token`  | `GITEA_TOKEN`               |
| Test group                                      | fluxcd-testing                | `GIT_PROVIDER_ORGANIZATION` |
| Test team (this is an ordinary group in GitLab) | fluxcd-testing-2              | `GITEA_TEST_TEAM_NAME`      |
| Test user                                       | fluxcd-gitprovider-bot        | `GITEA_USER`                |

## Continuous Integration

The e2e test suite runs in GitHub Actions on each commit to the main branch and on branches pushed to the repository, i.e. on PRs created from people with write access.

The provider configuration for the tests in CI deviates from the defaults listed above in the following ways:

### Stash

The tests are executed against a BitBucket Server hosted by Weaveworks. That server is maintained by @souleb.

| Setting   | Value                                                           |
| --------- | --------------------------------------------------------------- |
| Domain    | (please refer to @souleb if you really need to know this value) |
| Test user | fluxcd                                                          |
