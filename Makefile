SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=$(shell if `git diff-index --quiet HEAD --`; then echo false; else echo true;  fi)
LDFLAGS=-ldflags "-w -s -X github.com/chanzuckerberg/blessclient/pkg/util.GitSha=${SHA} -X github.com/chanzuckerberg/blessclient/pkg/util.Version=${VERSION} -X github.com/chanzuckerberg/blessclient/pkg/util.Dirty=${DIRTY}"
export GO111MODULE=on
export CGO_ENABLED=1

setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- v1.16.0
	curl -L https://raw.githubusercontent.com/chanzuckerberg/bff/master/download.sh | sh
	curl -sfL https://raw.githubusercontent.com/reviewdog/reviewdog/master/install.sh| sh -s -- v0.9.17
.PHONY: setup

test: deps ## run tests, will update go.mod
	go test -coverprofile=coverage.txt -covermode=atomic ./...
.PHONY: test

test-ci: ## run tests in CI, will fail if go.{mod,sum} are not up to date
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
.PHONY: deps

check-mod:
	go mod tidy
	git diff --exit-code -- go.mod go.sum
.PHONY: check-mod

lint: ## run the fast go linters
	./bin/golangci-lint run --no-config \
		--disable-all --enable=deadcode  --enable=gocyclo --enable=golint --enable=varcheck \
		--enable=structcheck --enable=errcheck --enable=dupl --enable=unparam --enable=goimports \
		--enable=interfacer --enable=unconvert --enable=gosec --enable=megacheck
.PHONY: lint

lint-ci: ## run the fast go linters
	./bin/reviewdog -tee -conf .reviewdog.yml  -reporter=github-pr-review
.PHONY: lint-ci

lint-all: ## run the fast go linters
	# doesn't seem to be a way to get reviewdog to not filter by diff
	./bin/golangci-lint run
.PHONY: lint-all
