.PHONY: test test-race test-cover test-integration integration-up integration-down mocks verify

mocks:
	GOFLAGS=-mod=vendor go generate -run mockgen ./internal/...

test: mocks
	go test -mod=vendor ./...

test-race: mocks
	CGO_ENABLED=1 go test -mod=vendor -race ./...

test-cover: mocks
	go test -mod=vendor -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-integration:
	go test -mod=vendor -tags=integration ./...

integration-up:
	docker compose up -d postgres redis

integration-down:
	docker compose down

verify:
	@echo "==> gofmt"
	@test -z "$$(gofmt -l ./internal/ ./cmd/ ./db/)" || (echo "gofmt check failed:"; gofmt -l ./internal/ ./cmd/ ./db/; exit 1)
	@echo "==> mocks"
	$(MAKE) mocks
	@echo "==> go vet"
	go vet -mod=vendor ./...
	@echo "==> go test"
	go test -mod=vendor ./...
	@echo "==> all checks passed"
