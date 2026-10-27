.PHONY: db-up migrate reset demo api web build start test

EXE := $(if $(filter Windows_NT,$(OS)),.exe,)

db-up:          ## Postgres en Docker
	docker compose up -d db

migrate:        ## Aplica las migraciones (incluye catálogos base)
	cd backend && go run ./cmd/api migrate

reset:          ## BORRA todo y deja la base lista para uso real (catálogos + cuentas del .env)
	cd backend && go run ./cmd/api reset --si

demo:           ## BORRA todo y carga datos de demostración (solo desarrollo)
	cd backend && go run ./cmd/api demo --si

api:            ## Backend en :8081 (desarrollo)
	cd backend && go run ./cmd/api

web:            ## Frontend con recarga en :5173 (desarrollo)
	cd frontend && npm run dev

build:          ## Compila frontend (frontend/dist) y backend (bin/api)
	cd frontend && npm ci && npm run build
	cd backend && go build -o ../bin/api$(EXE) ./cmd/api

start: ## Producción: la API sirve también la app en :8081
	cd backend && go build -o ../bin/api$(EXE) ./cmd/api
	cd backend && ../bin/api$(EXE)

test:
	cd backend && go test ./...
	cd frontend && npx tsc -b
