NAME = go-better-html-coverage

all: build

mkdir:
	mkdir -p bin

build: mkdir
	go build -o bin/$(NAME) ./

sanity: lint format test

lint:
	golangci-lint run --fix ./...

format:
	gofumpt -w .

test:
	go test ./...

coverage:
	go test ./... -covermode=count -coverprofile=coverage.out
	go tool cover -func=coverage.out -o=coverage.out

release:
	./scripts/make-release.sh
