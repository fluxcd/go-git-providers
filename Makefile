VER?=0.0.1

all: test

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

test: tidy fmt vet
	go test ./... -coverprofile cover.out

release:
	git checkout master
	git pull
	git tag "v$(VER)"
	git push origin "v$(VER)"
