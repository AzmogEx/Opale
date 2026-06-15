# Opale — raccourcis de développement.
.DEFAULT_GOAL := help
BACKEND := backend

.PHONY: help
help: ## Affiche cette aide
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## Met à jour go.mod/go.sum
	cd $(BACKEND) && go mod tidy

.PHONY: build
build: ## Compile le backend
	cd $(BACKEND) && go build ./...

.PHONY: test
test: ## Lance les tests (dont le moteur monétaire)
	cd $(BACKEND) && go test ./...

.PHONY: vet
vet: ## Analyse statique
	cd $(BACKEND) && go vet ./...

.PHONY: run
run: ## Lance le backend en local (nécessite Postgres, cf. make db)
	cd $(BACKEND) && go run ./cmd/api

.PHONY: db
db: ## Démarre uniquement PostgreSQL via Docker
	docker compose up -d db

.PHONY: up
up: ## Démarre toute la pile (db + api) via Docker
	docker compose up -d --build

.PHONY: down
down: ## Arrête la pile Docker
	docker compose down

.PHONY: logs
logs: ## Affiche les logs de l'API
	docker compose logs -f api

.PHONY: ci
ci: tidy vet test build ## Pipeline locale équivalente à la CI
