include .env
export

migrate-up:
	migrate -path db/migrations -database "postgres://clickit:clickit@localhost:5433/clickit?sslmode=disable" up

migrate-down:
	migrate -path db/migrations -database "postgres://clickit:clickit@localhost:5433/clickit?sslmode=disable" down 1

dev:
	make migrate-up
	go run .