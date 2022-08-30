# Development

> **Note:** Please take a look at <https://fluxcd.io/contributing/flux/>
> to find out about how to contribute to Flux and how to interact with the
> Flux Development team.

## How to run the test suite

Prerequisites:
* go >= 1.18

To run `make test` for your fork/branch you need to supply the following environment variables:

- `GIT_PROVIDER_ORGANIZATION` For GitHub this should be an organization; for GitLab this should be a top-level group; for BitBucket Server this should be project. As this environment variable is used for both test suites, the name of the GitHub organization must match the name of the GitLab top-level group. Also, this organization needs to have its repository default branch set to `main`.
- `GIT_PROVIDER_USER` This should be the same username for both GitHub and GitLab.
- `GITLAB_TEST_TEAM_NAME` An existing GitLab group.
- `GITLAB_TEST_SUBGROUP` An existing GitLab subgroup of the GIT_PROVIDER_ORGANIZATION top-level group.
- `GITPROVIDER_BOT_TOKEN` A GitHub token with `repo`, `admin:org` and `delete_repo` permissions.
- `GITLAB_TOKEN` A GitLab token with `api` scope.
- `STASH_USER` The BitBucket Server user name.
- `STASH_DOMAIN` The BitBucket Server domain name.
- `STASH_TOKEN` A BitBucket Server token.
- `TEST_VERBOSE` Set to '-v' to emit test output for debugging purposes
- `CLEANUP_ALL` Set to delete all test repos after testing.

## End-to-end testing

The e2e testing suite runs in GitHub Actions on each commit to the main branch.

The test suite targets the following providers:

* GitHub.com (fluxcd-testing organization)
* GitLab.com (fluxcd-testing group)
* BitBucket Server (hosted by Weaveworks in GCP)
