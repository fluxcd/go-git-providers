TEST_VERBOSE?=
TEST_PATTERN?=./...
TEST_STOP_ON_ERROR?=

all: test

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet
	go test ${TEST_VERBOSE} ${TEST_STOP_ON_ERROR} -race -coverprofile=coverage.txt -covermode=atomic ${TEST_PATTERN}
