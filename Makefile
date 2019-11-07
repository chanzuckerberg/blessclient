SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=$(shell if `git diff-index --quiet HEAD --`; then echo false; else echo true;  fi)
LDFLAGS=-ldflags "-w -s -X github.com/chanzuckerberg/blessclient/pkg/util.GitSha=${SHA} -X github.com/chanzuckerberg/blessclient/pkg/util.Version=${VERSION} -X github.com/chanzuckerberg/blessclient/pkg/util.Dirty=${DIRTY}"
export GOFLAGS=-mod=vendor
export GO111MODULE=on
export CGO_ENABLED=1

setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.16.0 # golangci-lint
	curl -L https://raw.githubusercontent.com/chanzuckerberg/bff/master/download.sh | BINDIR=./_bin sh
.PHONY: setup

test: deps ## run tests, will update vendor
	go test -coverprofile=coverage.txt -covermode=atomic ./...
.PHONY: test

test-ci: ## run tests in CI, will fail if vendor is not up to date
	go test -coverprofile=coverage.txt -covermode=atomic ./...
.PHONY: test

release: test ## Create a new tag and let travis_ci do the rest
	bff bump
	git push
	git push --tags
.PHONY: release

build: ## build the binary
	go build ${LDFLAGS} .
.PHONY: build

release-prerelease: test build ## release to github as a 'pre-release'
	version=`./blessclient version`; \
	git tag v"$$version"; \
	git push
	git push --tags
.PHONY: release-prerelease

publish-darwin:
	goreleaser release -f .goreleaser.darwin.yml --debug
.PHONY: publish-darwin

publish-prerelease-darwin:
	goreleaser release -f .goreleaser.prerelease.darwin.yml --debug
.PHONY: publish-prerelease-darwin

publish-linux: build ## Update the github release with a linux build. Must be run after release-darwin
	tar -zcf blessclient.tar.gz blessclient
	github-release upload \
	--user chanzuckerberg \
	--repo blessclient \
	--tag ${TRAVIS_TAG} \
	--name blessclient_${TRAVIS_TAG}_linux_amd64.tar.gz \
	--file blessclient.tar.gz
.PHONY: publish-linux

install: deps
	go install  ${LDFLAGS} .
.PHONY: install

deps:
	go mod tidy
	go mod vendor
.PHONY: deps

lint: ## run the fast go linters
	golangci-lint run --no-config \
		--disable-all --enable=deadcode  --enable=gocyclo --enable=golint --enable=varcheck \
		--enable=structcheck --enable=errcheck --enable=dupl --enable=unparam --enable=goimports \
		--enable=interfacer --enable=unconvert --enable=gosec --enable=megacheck
.PHONY: lint
