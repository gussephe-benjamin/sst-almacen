.PHONY: db-up migrate seed api web test

db-up:
	docker compose up -d db

migrate:
	cd backend && go run ./cmd/api migrate

seed:
	cd backend && go run ./cmd/api seed

api:
	cd backend && go run ./cmd/api

web:
	cd frontend && npm run dev

test:
	cd backend && go test ./...
