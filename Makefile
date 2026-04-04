.PHONY: up down logs build clean migrate-list auth-up auth-down auth-logs

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

build:
	for d in services/* tools/catalog-importer; do \
		( cd $$d && go build ./... ); \
	done

migrate-list:
	@find services -type f -path '*/migrations/*.sql' | sort

clean:
	docker compose -f deploy/docker-compose.yml --env-file deploy/.env down -v
