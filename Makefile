test:
	go test -cover ./...

install:
	go install .

.PHONY: test
