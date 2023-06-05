#!/usr/bin/env sh

# This script is intended to be used as a Docker HEALTHCHECK for the GitLab container.
# It prepares GitLab prior to running integration tests.
#
# This is a known workaround for docker-compose lacking lifecycle hooks.
# See: https://github.com/docker/compose/issues/1809#issuecomment-657815188

set -e

# Check for a successful HTTP status code from Gitea
curl --silent --show-error --fail --output /dev/null 127.0.0.1:3000

# Because this script runs on a regular health check interval,
# this file functions as a marker that tells us if initialization already finished.
done=/var/gitea-acctest-initialized

test -f $done || {
	echo 'Initializing Gitea for FluxCD integration tests'
  
  su git -s /bin/bash -c "gitea admin user create --admin --username $GITEA_USER --password $GITEA_USER --email admin@example.com"
  
  TOKEN=$(curl -H "Content-Type: application/json" -d '{"name":"fluxcd-1", "scopes":["sudo","repo","admin:org","admin:public_key","delete_repo"]}'  -u $GITEA_USER:$GITEA_USER http://127.0.0.1:3000/api/v1/users/$GITEA_USER/tokens \
  | sed -E 's/.*"sha1":"([^"]*).*/\1/')

  curl --silent --show-error --fail -v POST "http://127.0.0.1:3000/api/v1/admin/users/$GITEA_USER/orgs" \
  -H "Authorization: token $TOKEN" -H "Content-Type: application/json" \
  -H  "accept: application/json" \
  --data '{"full_name": "'"$GIT_PROVIDER_ORGANIZATION"'", "username": "'"$GIT_PROVIDER_ORGANIZATION"'"}'

  curl --silent --show-error --fail -v POST "http://127.0.0.1:3000/api/v1/orgs/$GIT_PROVIDER_ORGANIZATION/teams" \
  -H "Authorization: token $TOKEN" -H "Content-Type: application/json" \
  -H 'accept: application/json' \
  --data '{"name": "'"$GITEA_TEST_TEAM_NAME"'", "description": "'"$GITEA_TEST_TEAM_NAME"'", "permission": "read", "units": ["repo.code", "repo.issues", "repo.wiki", "repo.pulls"]}'

  touch $done
}

echo 'Gitea is ready for acceptance tests'
