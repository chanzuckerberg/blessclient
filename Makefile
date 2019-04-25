SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=$(shell if `git diff-index --quiet HEAD --`; then echo false; else echo true;  fi)
LDFLAGS=-ldflags "-w -s -X github.com/chanzuckerberg/blessclient/pkg/util.GitSha=${SHA} -X github.com/chanzuckerberg/blessclient/pkg/util.Version=${VERSION} -X github.com/chanzuckerberg/blessclient/pkg/util.Dirty=${DIRTY}"

setup:
	curl -L https://git.io/vp6lP | BINDIR=~/.local/bin sh # gometalinter
	curl -L https://raw.githubusercontent.com/chanzuckerberg/bff/master/download.sh | BINDIR=./_bin sh

test:
	go test -coverprofile=coverage.txt -covermode=atomic ./...

release:
	bff bump
	git push
	goreleaser release --rm-dist

build: ## build the binary
	go build ${LDFLAGS} .
.PHONY: build

release-prerelease: build ## release to github as a 'pre-release'
	version=`./blessclient version`; \
	git tag v"$$version"; \
	git push
	git push --tags
	goreleaser release -f .goreleaser.prerelease.yml --debug
.PHONY: release-prerelease

install:
	go install  ${LDFLAGS} .

lint: ## run the fast go linters
	gometalinter --vendor --fast --deadline=5m --disable=ineffassign ./...

.PHONY: test release install lint
