BINARY_NAME=curiosity-chat-api
MAIN_PATH=./cmd/api
GOBIN=$(shell go env GOPATH)/bin

DB_DSN=postgres://postgres:postgres@localhost:5434/curiosity_chat?sslmode=disable
DB_TEST_DSN=postgres://postgres:postgres@localhost:5435/curiosity_chat_test?sslmode=disable
MIGRATIONS_DIR=./migrations

.PHONY: help run build swag test test-short coverage fmt vet lint deps \
        migrate migrate-up migrate-down migrate-create migrate-version \
        migrate-up-test migrate-down-test \
        db-setup db-setup-test db-teardown db-teardown-test db-reset \
        install-tools

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

swag: ## Generate Swagger documentation
	$(GOBIN)/swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

build: swag ## Build the binary (generates docs first)
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)/main.go

run: swag build ## Generate docs, build, then run the API
	./bin/$(BINARY_NAME)

test: ## Run all tests with verbose output
	go test -v ./...

test-short: ## Run fast tests only
	go test -short -v ./...

coverage: ## Generate HTML coverage report
	go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

fmt: ## Format code
	gofmt -w .

vet: ## Run go vet
	go vet ./...

lint: fmt vet ## Format and vet

deps: ## Download and tidy dependencies
	go mod download && go mod tidy

migrate: migrate-up migrate-up-test ## Run migrations on both prod and test DBs

migrate-up: ## Run pending migrations on prod DB
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

migrate-down: ## Rollback last migration on prod DB
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1

migrate-create: ## Create a new migration (NAME=migration_name)
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)

migrate-version: ## Show current migration version
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" version

migrate-up-test: ## Run migrations on test DB
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_TEST_DSN)" up

migrate-down-test: ## Rollback on test DB
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_TEST_DSN)" down 1

db-setup: ## Start postgres and migrate
	docker compose up -d postgres
	@echo "Waiting for postgres..." && sleep 3
	$(MAKE) migrate-up

db-setup-test: ## Start test postgres and migrate
	docker compose up -d postgres_test
	@echo "Waiting for test postgres..." && sleep 3
	$(MAKE) migrate-up-test

db-teardown: ## Stop and remove containers
	docker compose down -v

db-teardown-test: ## Stop test container
	docker compose stop postgres_test && docker compose rm -f postgres_test

db-reset: ## Reset both databases
	$(MAKE) db-teardown
	$(MAKE) db-setup
	$(MAKE) db-setup-test

install-tools: ## Install required CLI tools (migrate, swag)
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/swaggo/swag/cmd/swag@latest
