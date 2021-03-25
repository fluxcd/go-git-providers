VER?=0.0.1
TEST_VERBOSE?=

all: test

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet
	go test ${TEST_VERBOSE} -race -coverprofile=coverage.txt -covermode=atomic ./...

release:
	git checkout master
	git pull
	git tag "v$(VER)"
	git push origin "v$(VER)"
