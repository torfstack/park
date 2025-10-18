## park justfile

lint:
    golangci-lint run ./...

fmt:
    golangci-lint fmt ./...

dep-update:
    go get -u ./...
    go mod tidy