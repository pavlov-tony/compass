.PHONY: generate
generate:
	@echo "Running go generate..."
	@go generate ./...

.PHONY: proto
proto:
	@echo "Generating Protobuf files..."
	@buf generate

.PHONY: test
test: generate
	@echo "Running tests..."
	@go test -v -cover ./...

.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker build -t compass .

.PHONY: build-binary
build-binary: docker-build
	@echo "Extracting binary from container..."
	@mkdir -p bin
	@docker create --name temp-compass compass
	@docker cp temp-compass:/usr/bin/coredns ./bin/coredns
	@docker rm temp-compass
	@echo "Done. Binary is at ./bin/coredns"
	@echo "You can verify it with: ./bin/coredns -plugins"
