.PHONY: up down build test run

up:
	docker-compose up -d

down:
	docker-compose down -v

build:
	go build -o bin/example ./cmd/example

test:
	go test -v ./...

run: up
	go run ./cmd/example 