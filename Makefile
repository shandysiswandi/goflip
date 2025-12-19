.PHONY: run
run:
	@LOCAL=true go run main.go

.PHONY: test
test:
	@go test ./...

.PHONY: test-race
test-race:
	@go test -race ./...

.PHONY: lint
lint:
	@golangci-lint run
