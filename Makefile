.PHONY: generate
generate:
	go generate ./...

.PHONY: test
test: generate
	go test -v -cover ./...

.PHONY: docker-build
docker-build:
	docker build -t compass .

.PHONY: build-binary
build-binary: docker-build
	@echo "Extracting binary from container..."
	@mkdir -p bin
	@docker create --name temp-compass compass
	@docker cp temp-compass:/usr/bin/coredns ./bin/coredns
	@docker rm temp-compass
	@echo "Done. Binary is at ./bin/coredns"
	@echo "You can verify it with: ./bin/coredns -plugins"
