SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=$(shell if `git diff-index --quiet HEAD --`; then echo false; else echo true;  fi)
LDFLAGS=-ldflags "-w -s -X github.com/chanzuckerberg/blessclient/pkg/util.GitSha=${SHA} -X github.com/chanzuckerberg/blessclient/pkg/util.Version=${VERSION} -X github.com/chanzuckerberg/blessclient/pkg/util.Dirty=${DIRTY}"

test:
	go test -cover ./...

release:
	./release
	goreleaser release --rm-dist
	git push

install:
	go install  ${LDFLAGS} .

.PHONY: test release install
