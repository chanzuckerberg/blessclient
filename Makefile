test:
	go test -cover ./...

release:
	goreleaser release --rm-dist

install:
	go install .

.PHONY: test release install
