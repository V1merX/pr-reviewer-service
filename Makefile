.PHONY: all help build run tests coverage short-tests fmt lint tidy \
		docker_build docker_up docker_down docker_logs docker_prune \
		clean dev migrate ci

OS_NAME := $(shell uname -s 2>/dev/null || echo Windows)

APP := pr-reviewer-app
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP)

GO := go
GOFLAGS := -v
IMAGE := pr-reviewer:latest

ensure-bin := $(shell mkdir -p $(BIN_DIR) 2>/dev/null || true)

help:
	@printf "Service helper - available make targets:\n"
	@printf "  build           Build the service binary\n"
	@printf "  run             Build and run the binary\n"
	@printf "  dev             Run with live reload (requires 'air')\n"
	@printf "  tests           Run all tests\n"
	@printf "  coverage        Run tests and produce coverage report\n"
	@printf "  fmt             Format sources and fix imports\n"
	@printf "  lint            Lint the codebase with golangci-lint\n"
	@printf "  tidy            Tidy go modules\n"
	@printf "  docker_up       Start services via docker-compose\n"
	@printf "  docker_down     Stop services\n"
	@printf "  docker_logs     Stream compose logs\n"
	@printf "  migrate         Apply DB migrations\n"
	@printf "  clean           Remove generated files\n"
	@printf "  ci              Run lint + tests\n"

build: $(ensure-bin)
	@echo "--> building: $(APP)"
	$(GO) build $(GOFLAGS) -o $(BIN) ./cmd/api
	@echo "--> built: $(BIN)"

run: build
	@echo "--> launching: $(BIN)"
	@./$(BIN)

tests:
	@echo "--> running: unit tests"
	$(GO) test $(GOFLAGS) ./...

coverage:
	@echo "--> running: tests with coverage"
	$(GO) test -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "--> coverage written to coverage.html"

short-tests:
	@echo "--> running: short tests"
	$(GO) test -short ./...

fmt:
	@echo "--> formatting sources"
	$(GO) fmt ./...
	@which goimports >/dev/null 2>&1 || $(GO) install golang.org/x/tools/cmd/goimports@latest
	@goimports -w . || true
	@echo "--> format complete"

lint:
	@echo "--> lint: checking with golangci-lint"
	@which golangci-lint >/dev/null 2>&1 || $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run ./... || true

tidy:
	@echo "--> tidy: go modules"
	$(GO) mod tidy
	$(GO) mod verify || true
	@echo "--> tidy complete"

# Docker helpers
docker_build:
	@echo "--> docker: building image $(IMAGE)"
	docker build -t $(IMAGE) .
	@echo "--> docker image ready"

docker_up:
	@echo "--> docker: bringing services up"
	docker-compose up -d --build
	@echo "--> docker: services started"
	docker-compose logs -f

docker_down:
	@echo "--> docker: stopping services"
	docker-compose down
	@echo "--> docker: stopped"

docker_logs:
	@echo "--> docker: streaming logs"
	docker-compose logs -f

docker_prune:
	@echo "--> docker: stop + cleanup"
	docker-compose down -v
	-@docker rmi $(IMAGE) 2>/dev/null || true
	@echo "--> docker: cleanup finished"

	@echo "--> running DB migration"
	psql -U postgres -d pr_review_db -f migrations/001_init.sql || true
	@echo "--> migration finished"

clean:
	@echo "--> cleaning build outputs"
	-@rm -rf $(BIN_DIR) coverage.out coverage.html || true
	@$(GO) clean || true
	@echo "--> clean complete"

dev:
	@echo "--> starting dev mode (air)"
	@which air >/dev/null 2>&1 || $(GO) install github.com/cosmtrek/air@latest
	@air || true

ci: lint tests
	@echo "--> CI tasks done"

test-coverage: coverage
test-short: short-tests
docker-clean: docker_prune
migrate-up: migrate
	docker-compose down

