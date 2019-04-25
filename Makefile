SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=$(shell if `git diff-index --quiet HEAD --`; then echo false; else echo true;  fi)
LDFLAGS=-ldflags "-w -s -X github.com/chanzuckerberg/blessclient/pkg/util.GitSha=${SHA} -X github.com/chanzuckerberg/blessclient/pkg/util.Version=${VERSION} -X github.com/chanzuckerberg/blessclient/pkg/util.Dirty=${DIRTY}"

setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.16.0 # golangci-lint
	curl -L https://raw.githubusercontent.com/chanzuckerberg/bff/master/download.sh | BINDIR=./_bin sh
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh

test:
	go test -coverprofile=coverage.txt -covermode=atomic ./...

release: test ## Create a new tag and let travis_ci do the rest
	bff bump
	git push

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

publish-prerelease-darwin:
	goreleaser release -f .goreleaser.prerelease.darwin.yml --debug

publish-linux: build ## Update the github release with a linux build. Must be run after release-darwin
	tar -zcvf blessclient.tar.gz blessclient
	github-release upload \
	--security_token ${GITHUB_TOKEN} \
	--user czibuildbot \
	--repo blessclient \
	--tag ${TRAVIS_TAG} \
	--name blessclient_${TRAVIS_TAG}_linux_amd64 \
	--file blessclient.tar.gz


install:
	go install  ${LDFLAGS} .

lint: ## run the fast go linters
	golangci-lint run --no-config \
		--disable-all --enable=deadcode  --enable=gocyclo --enable=golint --enable=varcheck \
		--enable=structcheck --enable=errcheck --enable=dupl --enable=unparam --enable=goimports \
		--enable=interfacer --enable=unconvert --enable=gosec --enable=megacheck

.PHONY: test release install lint
