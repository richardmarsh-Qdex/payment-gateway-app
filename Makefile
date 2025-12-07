.PHONY: build run test clean docker-build docker-up docker-down migrate

# Build the application
build:
	go build -o payment-gateway main.go

# Run the application
run:
	go run main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -f payment-gateway
	rm -f coverage.out

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Docker build
docker-build:
	docker build -t payment-gateway:latest .

# Docker compose up
docker-up:
	docker-compose up -d

# Docker compose down
docker-down:
	docker-compose down

# Run migrations (if needed)
migrate:
	go run main.go migrate

# Install dependencies
deps:
	go mod download
	go mod tidy

# Generate API docs (if using swag)
docs:
	swag init

