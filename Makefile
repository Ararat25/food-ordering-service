.PHONY: build run test lint migrate seed docker-up docker-down

build:
	go build -o bin/server ./cmd/server
	go build -o bin/venue-server ./cmd/venue

run:
	go run ./cmd/server

run-venue:
	go run ./cmd/venue

test:
	go test -v ./...

lint:
	golangci-lint run ./...

migrate-up:
	@set -a; if [ -f .env ]; then . ./.env; fi; set +a; goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	@set -a; if [ -f .env ]; then . ./.env; fi; set +a; goose -dir migrations postgres "$$DATABASE_URL" down

seed: migrate-up

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

clean:
	rm -rf bin/
	docker compose down -v
