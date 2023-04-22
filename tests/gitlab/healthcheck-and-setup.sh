#!/usr/bin/env sh

# This script is intended to be used as a Docker HEALTHCHECK for the GitLab container.
# It prepares GitLab prior to running integration tests.
#
# This is a known workaround for docker-compose lacking lifecycle hooks.
# See: https://github.com/docker/compose/issues/1809#issuecomment-657815188

set -e

# Check for a successful HTTP status code from GitLab.
curl --silent --show-error --fail --output /dev/null 127.0.0.1:80

# Because this script runs on a regular health check interval,
# this file functions as a marker that tells us if initialization already finished.
done=/var/gitlab-acctest-initialized

test -f $done || {
	echo 'Initializing GitLab for FluxCD integration tests'
	gitlab-rails console <<EOF
git_provider_user = User.new(name: '$GIT_PROVIDER_USER', username: '$GIT_PROVIDER_USER', email: '$GIT_PROVIDER_USER@example.com', password: 'testtesttest', confirmed_at: Time.now, can_create_group: true)
git_provider_user.save!

git_provider_user_pat = PersonalAccessToken.create(user_id: git_provider_user.id, scopes: [:api, :read_user], name: 'go-git-provider')
git_provider_user_pat.set_token('$GITLAB_TOKEN')
git_provider_user_pat.save!

# Group and Project setup
git_provider_organization = ::Groups::CreateService.new(git_provider_user, { name: '$GIT_PROVIDER_ORGANIZATION', path: '$GIT_PROVIDER_ORGANIZATION' }).execute
::Groups::CreateService.new(git_provider_user, { name: '$GITLAB_TEST_SUBGROUP', path: '$GITLAB_TEST_SUBGROUP', parent_id: git_provider_organization.id }).execute
::Projects::CreateService.new(git_provider_user, { name: '$GITLAB_TEST_REPO_NAME', namespace_id: git_provider_organization.id }).execute
::Groups::CreateService.new(git_provider_user, { name: '$GITLAB_TEST_TEAM_NAME', path: '$GITLAB_TEST_TEAM_NAME' }).execute
EOF

	touch $done
}

echo 'GitLab is ready for acceptance tests'
