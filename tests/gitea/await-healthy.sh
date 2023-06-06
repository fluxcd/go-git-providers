#!/bin/bash

set -e

printf 'Waiting for Gitea container to become healthy'

until test -n "$(docker ps --quiet --filter label=go-git-provider-gitea/owned --filter health=healthy)"; do
	printf '.'
	sleep 5
done

echo
echo "Gitea is healthy at $GITEA_BASE_URL"

export GITEA_TOKEN=$(curl -H "Content-Type: application/json" -d '{"name":"fluxcd-2", "scopes":["sudo","repo","admin:org","admin:public_key","delete_repo"]}' -u $GITEA_USER:$GITEA_USER $GITEA_BASE_URL/api/v1/users/$GITEA_USER/tokens \
| sed -E 's/.*"sha1":"([^"]*).*/\1/')

# Print the version, since it is useful debugging information.
curl --silent --show-error --header "Authorization: token $GITEA_TOKEN" "$GITEA_BASE_URL/api/v1/version"
echo

# Keep token in tmp file for later use
echo $GITEA_TOKEN > /tmp/gitea-token

echo "Gitea token saved to /tmp/gitea-token"
