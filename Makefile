.DEFAULT_GOAL := help
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: help test cover lint vuln build run demo record bench

help: ## list targets
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-8s %s\n",$$1,$$2}'

test: ## vet + race tests
	go vet ./... && go test -race ./...

cover: ## coverage report
	go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

lint: ## golangci-lint (install: https://golangci-lint.run)
	golangci-lint run

vuln: ## known-vulnerability scan
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

build: ## static binary with version stamp
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o qde ./cmd/qde

run: ## one pass (live if keys are set, otherwise the committed recording)
	go run ./cmd/qde

demo: ## paced output for screen recording
	go run ./cmd/qde --delay 900ms

record: ## call the live API once and rewrite data/replay/clef-flash.json
	go run ./cmd/qde --mode live --record

bench: ## latency over 200 live calls; set CLEF_URL for a self-hosted server
	go run ./cmd/qde --bench 200
