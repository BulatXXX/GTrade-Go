.PHONY: up down logs build clean migrate-list

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

build:
	for d in services/* tools/catalog-importer; do \
		( cd $$d && go build ./... ); \
	done

migrate-list:
	@find services -type f -path '*/migrations/*.sql' | sort

clean:
	docker compose down -v
