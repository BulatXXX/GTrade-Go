.PHONY: up down logs build clean migrate-list auth-up auth-down auth-logs auth-db-up auth-db-down auth-test auth-test-integration auth-notification-up auth-notification-down auth-notification-logs auth-notification-e2e-test notification-up notification-down notification-logs notification-db-up notification-db-down notification-test notification-test-integration catalog-up catalog-down catalog-logs catalog-db-up catalog-db-down catalog-test catalog-test-integration catalog-build catalog-backup catalog-restore

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

auth-notification-up:
	EMAIL_PROVIDER=mock docker compose -f deploy/docker-compose.yml --env-file deploy/.env up --build -d postgres-auth postgres-notification notification-service auth-service

auth-notification-down:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env rm -sf auth-service notification-service postgres-auth postgres-notification

auth-notification-logs:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env logs -f postgres-auth postgres-notification notification-service auth-service

auth-notification-e2e-test:
	@if [ ! -f deploy/.env ]; then echo "deploy/.env is missing. Run: cp deploy/.env.example deploy/.env"; exit 1; fi
	EMAIL_PROVIDER=mock docker compose -f deploy/docker-compose.yml --env-file deploy/.env up --build -d postgres-auth postgres-notification notification-service auth-service
	cd services/auth-service && AUTH_BASE_URL='http://localhost:8081' NOTIFICATION_TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' GOCACHE=/tmp/gocache-auth go test ./internal/e2e -v

notification-up:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up --build -d postgres-notification notification-service

notification-down:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env rm -sf notification-service postgres-notification

notification-logs:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env logs -f postgres-notification notification-service

notification-db-up:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d postgres-notification

notification-db-down:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env rm -sf postgres-notification

notification-test:
	@if [ ! -f deploy/.env ]; then echo "deploy/.env is missing. Run: cp deploy/.env.example deploy/.env"; exit 1; fi
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d postgres-notification
	cd services/notification-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' GOCACHE=/tmp/gocache-notification go test ./...

notification-test-integration:
	@if [ ! -f deploy/.env ]; then echo "deploy/.env is missing. Run: cp deploy/.env.example deploy/.env"; exit 1; fi
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d postgres-notification
	cd services/notification-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5437/gtrade_notification?sslmode=disable' GOCACHE=/tmp/gocache-notification go test ./internal/http -run TestSendEmailIntegration -v

catalog-up:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up --build -d postgres-catalog catalog-service

catalog-down:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env rm -sf catalog-service postgres-catalog

catalog-logs:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env logs -f postgres-catalog catalog-service

catalog-db-up:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d postgres-catalog

catalog-db-down:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env rm -sf postgres-catalog

catalog-test:
	@if [ ! -f deploy/.env ]; then echo "deploy/.env is missing. Run: cp deploy/.env.example deploy/.env"; exit 1; fi
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d postgres-catalog
	cd services/catalog-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5436/gtrade_catalog?sslmode=disable' GOCACHE=/tmp/gocache-catalog go test ./...

catalog-test-integration:
	@if [ ! -f deploy/.env ]; then echo "deploy/.env is missing. Run: cp deploy/.env.example deploy/.env"; exit 1; fi
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d postgres-catalog
	cd services/catalog-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5436/gtrade_catalog?sslmode=disable' GOCACHE=/tmp/gocache-catalog go test ./internal/repository -v

catalog-build:
	cd services/catalog-service && go build ./...

catalog-backup:
	@mkdir -p backups
	@backup_file=$${BACKUP_FILE:-backups/catalog-$$(date +%Y%m%d-%H%M%S).dump}; \
	echo "Creating catalog backup: $$backup_file"; \
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
		pg_dump -U gtrade -d gtrade_catalog -Fc > "$$backup_file"

catalog-restore:
	@if [ -z "$(BACKUP_FILE)" ]; then echo "BACKUP_FILE is required. Example: make catalog-restore BACKUP_FILE=backups/catalog-20260405-013000.dump"; exit 1; fi
	@cat "$(BACKUP_FILE)" | docker compose -f deploy/docker-compose.yml --env-file deploy/.env exec -T postgres-catalog \
		pg_restore -U gtrade -d gtrade_catalog --clean --if-exists

build:
	for d in services/* tools/catalog-importer; do \
		( cd $$d && go build ./... ); \
	done

migrate-list:
	@find services -type f -path '*/migrations/*.sql' | sort

clean:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env down -v
