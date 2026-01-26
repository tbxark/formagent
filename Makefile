.PHONY: format
format:
	golangci-lint fmt --no-config --enable gofmt,goimports
	golangci-lint run --no-config --fix
	go fmt ./...
	go mod tidy