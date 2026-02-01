.PHONY: test test-race test-cover test-integration integration-up integration-down

test:
	go test -mod=vendor ./...

test-race:
	CGO_ENABLED=1 go test -mod=vendor -race ./...

test-cover:
	go test -mod=vendor -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-integration:
	go test -mod=vendor -tags=integration ./...

integration-up:
	docker compose up -d postgres redis

integration-down:
	docker compose down
