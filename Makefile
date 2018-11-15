SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=$(shell if `git diff-index --quiet HEAD --`; then echo false; else echo true;  fi)
LDFLAGS=-ldflags "-w -s -X github.com/chanzuckerberg/blessclient/pkg/util.GitSha=${SHA} -X github.com/chanzuckerberg/blessclient/pkg/util.Version=${VERSION} -X github.com/chanzuckerberg/blessclient/pkg/util.Dirty=${DIRTY}"

setup:
	curl -L https://git.io/vp6lP | BINDIR=~/.local/bin sh # gometalinter

test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

release:
	./_bin/release
	git push
	goreleaser release --rm-dist

install:
	go install  ${LDFLAGS} .

lint: ## run the fast go linters
	gometalinter --vendor --fast --deadline=5m --disable=ineffassign ./...

.PHONY: test release install lint
