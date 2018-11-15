SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=$(shell if `git diff-index --quiet HEAD --`; then echo false; else echo true;  fi)
LDFLAGS=-ldflags "-w -s -X github.com/chanzuckerberg/blessclient/pkg/util.GitSha=${SHA} -X github.com/chanzuckerberg/blessclient/pkg/util.Version=${VERSION} -X github.com/chanzuckerberg/blessclient/pkg/util.Dirty=${DIRTY}"

setup:
	go get -u golang.org/x/lint/golint

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

release:
	./_bin/release
	git push
	goreleaser release --rm-dist

install:
	go install  ${LDFLAGS} .

lint: ## run the go linters
	@golint -set_exit_status $(shell go list ./... | grep -v /vendor/)

.PHONY: test release install lint
