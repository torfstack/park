## park justfile

lint:
    golangci-lint run ./...

fmt:
    golangci-lint fmt ./...

dep-update:
    go get -u ./...
    go mod tidy

[working-directory: 'cmd/park']
install:
    go install .

test:
    go test ./...