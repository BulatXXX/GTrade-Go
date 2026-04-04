.PHONY: up down logs build clean migrate-list auth-up auth-down auth-logs auth-db-up auth-db-down auth-test auth-test-integration

up:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up --build -d

down:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env down

logs:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env logs -f

auth-up:
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env up --build -d

auth-down:
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env down -v

auth-logs:
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env logs -f

auth-db-up:
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env up -d postgres-auth

auth-db-down:
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env down -v

auth-test:
	@if [ ! -f deploy/.env ]; then echo "deploy/.env is missing. Run: cp deploy/.env.example deploy/.env"; exit 1; fi
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env up -d postgres-auth
	cd services/auth-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5433/gtrade_auth?sslmode=disable' GOCACHE=/tmp/gocache-auth go test ./...

auth-test-integration:
	@if [ ! -f deploy/.env ]; then echo "deploy/.env is missing. Run: cp deploy/.env.example deploy/.env"; exit 1; fi
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env up -d postgres-auth
	cd services/auth-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5433/gtrade_auth?sslmode=disable' GOCACHE=/tmp/gocache-auth go test ./internal/service -run TestAuthServiceIntegration -v

build:
	for d in services/* tools/catalog-importer; do \
		( cd $$d && go build ./... ); \
	done

migrate-list:
	@find services -type f -path '*/migrations/*.sql' | sort

clean:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env down -v
