.PHONY: generate test

generate:
	go generate ./...

test: generate
	go test -v -cover ./...
