# GTrade Data System (GTrade)

GTrade Data System — платформа управления данными внутриигровых торговых площадок.

## Структура репозитория

- `services/` — Go-микросервисы
- `tools/` — Go CLI-утилиты
- `frontend/` — заглушка под Angular
- `docs/` — документация по архитектуре и сервисам

## Сервисы

- `api-gateway`
- `auth-service`
- `user-asset-service`
- `marketplace-integration-service`
- `catalog-service`
- `notification-service`
- `tools/catalog-importer`

## Технологический стек

- Go
- Gin
- PostgreSQL
- pgx pool
- zerolog
- Docker / docker-compose

## Запуск локально

1. Скопировать env-файл:

```bash
cp .env.example .env
```

2. Поднять инфраструктуру:

```bash
make up
```

3. Проверить health endpoint'ы:

- `http://localhost:8080/health`
- `http://localhost:8081/health`
- `http://localhost:8082/health`
- `http://localhost:8083/health`
- `http://localhost:8084/health`
- `http://localhost:8085/health`

4. Остановить:

```bash
make down
```

## Порты сервисов

- api-gateway: `8080`
- auth-service: `8081`
- user-asset-service: `8082`
- marketplace-integration-service: `8083`
- catalog-service: `8084`
- notification-service: `8085`

## Порты PostgreSQL

- auth DB: `5433`
- user-asset DB: `5434`
- catalog DB: `5436`
- notification DB: `5437`

`api-gateway` и `marketplace-integration-service` в текущем skeleton работают без собственной БД.

## Что уже реализовано

- production-like skeleton для всех сервисов и утилиты
- единый каркас: конфиг из env, логирование, HTTP, repository layer
- health endpoint во всех сервисах
- заглушки endpoint'ов по доменам
- middleware логирования запросов
- JWT middleware-заглушка в `api-gateway`
- абстракции адаптеров маркетплейсов
- абстракции провайдеров уведомлений (mock + resend skeleton)
- docker-compose для локальной разработки на Mac

## Что пока заглушка

- реальная бизнес-логика
- реальная JWT-аутентификация (выдача/проверка токенов)
- реальный reverse proxy / service client flow в gateway
- интеграции с внешними API маркетплейсов
- полноценная интеграция с Resend
- frontend на Angular

## Подход к данным

БД оставлены только в сервисах, где хранится состояние в текущем этапе:
`auth-service`, `user-asset-service`, `catalog-service`, `notification-service`.
