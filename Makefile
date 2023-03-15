TEST_FLAGS?=
TEST_PATTERN?=./...

all: test

tidy:
	go mod tidy -compat=1.18

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet
	go test ${TEST_FLAGS} -race -coverprofile=coverage.txt -covermode=atomic ${TEST_PATTERN}
