# Make Targets Template

Шаблон для добавления `make`-команд под отдельный сервис.

## Базовые правила

- каждая команда должна быть привязана к одному сервису
- префикс команды должен совпадать с именем сервиса или коротким алиасом
- команды должны быть предсказуемыми и одинаковыми между сервисами
- если для сервиса нужна БД, должны быть отдельные команды для поднятия только БД
- если для сервиса есть интеграционные тесты, они должны запускаться одной командой

## Рекомендуемая схема именования

Для сервиса `<service>`:

```make
<service>-up
<service>-down
<service>-logs
<service>-db-up
<service>-db-down
<service>-test
<service>-test-integration
<service>-build
```

## Назначение команд

- `<service>-up`:
  поднимает сервис и его минимальные зависимости для ручной проверки

- `<service>-down`:
  останавливает сервисный compose

- `<service>-logs`:
  показывает логи сервиса и связанных контейнеров

- `<service>-db-up`:
  поднимает только БД сервиса, если она нужна

- `<service>-db-down`:
  останавливает и при необходимости очищает только БД сервиса

- `<service>-test`:
  запускает все тесты сервиса одной командой

- `<service>-test-integration`:
  запускает только интеграционные тесты с реальной инфраструктурой

- `<service>-build`:
  собирает сервис локально без запуска всего монорепо

## Пример для auth-service

```make
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
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env up -d postgres-auth
	cd services/auth-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5433/gtrade_auth?sslmode=disable' GOCACHE=/tmp/gocache-auth go test ./...

auth-test-integration:
	docker compose -f deploy/docker-compose.auth.yml --env-file deploy/.env up -d postgres-auth
	cd services/auth-service && TEST_DATABASE_URL='postgres://gtrade:gtrade@localhost:5433/gtrade_auth?sslmode=disable' GOCACHE=/tmp/gocache-auth go test ./internal/service -run TestAuthServiceIntegration -v

auth-build:
	cd services/auth-service && go build ./...
```

## Чеклист перед добавлением новых команд

- команда работает из корня репозитория
- команда не требует ручного выставления переменных, если можно подставить их внутри `Makefile`
- команда не ломается без уже поднятых контейнеров
- команда отражена в документации сервиса
- имя команды понятно без чтения реализации
