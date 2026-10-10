include .env

run:
	@go run ./...

test:
	@go test ./...

migrate-up:
	@migrate -path migrations -database $(DATABASE_URL) up

migrate-down:
	@migrate -path migrations -database $(DATABASE_URL) down
