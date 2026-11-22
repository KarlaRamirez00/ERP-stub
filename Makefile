.PHONY: start test build docker

start:
	go run ./cmd

test:
	go vet ./...
	go test ./...

build:
	go build -trimpath -o bin/rex-stub ./cmd

docker:
	docker compose up --build
