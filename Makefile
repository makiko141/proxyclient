.PHONY: lint tests
lint:
	golangci-lint run

tests:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...