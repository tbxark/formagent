.PHONY: format
format:
	golangci-lint fmt --no-config --enable gofmt,goimports
	golangci-lint run --no-config --fix
	go fmt ./...
	go mod tidy

.PHONY: test
test:
	FORMAGENT_RUN_LIVE_TESTS=1 go test ./testcases -v -parallel 2