test:
	go test -cover ./...

release:
	goreleaser release --rm-dist

.PHONY: test
