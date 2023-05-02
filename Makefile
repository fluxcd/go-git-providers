TEST_FLAGS?=
TEST_PATTERN?=./...

all: test

tidy:
	go mod tidy -compat=1.18

fmt:
	go fmt ./...

vet:
	go vet ./...

GIT_PROVIDER_ORGANIZATION ?= fluxcd-testing
GIT_PROVIDER_USER ?= fluxcd-gitprovider-bot

# GitLab specific testing variables
GITLAB_TOKEN ?= glpat-ACCTEST1234567890123
GITLAB_BASE_URL ?= http://127.0.0.1:9042
GITLAB_TEST_REPO_NAME ?= fluxcd-testing-repo
GITLAB_TEST_SUBGROUP ?= fluxcd-testing-sub-group
GITLAB_TEST_TEAM_NAME ?= fluxcd-testing-2

start-provider-instances-gitlab:
	GITLAB_TOKEN=$(GITLAB_TOKEN) GIT_PROVIDER_USER=$(GIT_PROVIDER_USER) GIT_PROVIDER_ORGANIZATION=$(GIT_PROVIDER_ORGANIZATION) GITLAB_TEST_REPO_NAME=$(GITLAB_TEST_REPO_NAME) GITLAB_TEST_SUBGROUP=$(GITLAB_TEST_SUBGROUP) GITLAB_TEST_TEAM_NAME=$(GITLAB_TEST_TEAM_NAME) docker compose up -d gitlab
	GITLAB_BASE_URL=$(GITLAB_BASE_URL) GITLAB_TOKEN=$(GITLAB_TOKEN) ./tests/gitlab/await-healthy.sh

start-provider-instances: start-provider-instances-gitlab

stop-provider-instances:
	docker compose down --volumes

test: tidy fmt vet
	go test ${TEST_FLAGS} -race -coverprofile=coverage.txt -covermode=atomic ${TEST_PATTERN}

test-e2e-github: tidy fmt vet
	go test ${TEST_FLAGS} -race -coverprofile=coverage.txt -covermode=atomic -tags=e2e ./github/...

test-e2e-gitlab: tidy fmt vet
	GITLAB_BASE_URL=$(GITLAB_BASE_URL) GITLAB_TOKEN=$(GITLAB_TOKEN) go test ${TEST_FLAGS} -race -coverprofile=coverage.txt -covermode=atomic -tags=e2e ./gitlab/...

test-e2e-stash: tidy fmt vet
	go test ${TEST_FLAGS} -race -coverprofile=coverage.txt -covermode=atomic -tags=e2e ./stash/...

