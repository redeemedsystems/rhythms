.PHONY: build run test tidy

build:
	CGO_ENABLED=0 go build -o bin/rhythms ./cmd/rhythms

run: build
	./bin/rhythms

test:
	go test ./...

tidy:
	go mod tidy
