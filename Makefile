.PHONY: test test-race test-cover

test:
	go test -mod=vendor ./...

test-race:
	CGO_ENABLED=1 go test -mod=vendor -race ./...

test-cover:
	go test -mod=vendor -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
