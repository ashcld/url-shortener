.PHONY: run build test lint migrate-up migrate-down migrate-create proto docker-up docker-down

BINARY=bin/api
WORKER=bin/worker
MIGRATIONS_PATH=./migrations
DATABASE_URL=postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)

include .env
export

run:
	go run ./cmd/api/...

run-worker:
	go run ./cmd/worker/...

build:
	go build -o $(BINARY) ./cmd/api/...
	go build -o $(WORKER) ./cmd/worker/...

test:
	go test -race -count=1 ./...

test-cover:
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/urlshortener/url.proto

docker-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-down:
	docker compose -f deployments/docker-compose.yml down

docker-dev:
	docker compose -f deployments/docker-compose.dev.yml up -d
