# auth-service

Сервис аутентификации GTrade.

## Что уже готово

- HTTP-сервер на Gin
- загрузка конфигурации из env
- подключение к PostgreSQL через repository layer
- хранение пользователей в `users`
- хранение refresh token'ов в `refresh_tokens`
- хранение password reset token'ов в `password_reset_tokens`
- хранение email verification token'ов в `email_verification_tokens`
- bcrypt-хеширование паролей
- JWT access/refresh token flow
- password reset flow
- email verification flow
- интеграция `auth-service` с `notification-service`
- smoke tests HTTP-слоя
- интеграционные тесты с реальной PostgreSQL
- локальная документация и OpenAPI в `docs/`

## Готовые endpoint'ы

- `GET /health`
- `POST /register`
- `POST /login`
- `POST /refresh`
- `POST /password/reset/request`
- `POST /password/reset/confirm`
- `POST /email/verify`

## Текущий MVP flow

- `POST /password/reset/request` создает reset token, сохраняет его в БД и отправляет его через `notification-service`
- `POST /password/reset/confirm` принимает reset token и новый пароль, затем меняет пароль
- `POST /email/verify` в режиме request создает verification token и отправляет его через `notification-service`
- `POST /email/verify` в режиме confirm принимает verification token и помечает email как подтвержденный

## Что нужно доделать

- разделение request/confirm сценариев на более явный публичный контракт при необходимости
- валидация email и password на уровне входных DTO
- защита от user enumeration в reset/verify request flow
- rate limiting / throttling для чувствительных auth endpoint'ов
- logout / revoke flow
- account endpoint'ы вроде `GET /me` при необходимости
- swagger-ui или другой способ локально просматривать OpenAPI

## Ключевые файлы

- `internal/service/auth.go` — основная бизнес-логика auth flow
- `internal/repository/auth_repository.go` — работа с PostgreSQL
- `internal/handler/auth.go` — HTTP handlers
- `internal/model/auth.go` — request/response DTO
- `internal/http/service_routes.go` — маршруты сервиса
- `migrations/0001_init.sql` — базовая auth schema
- `migrations/0002_account_lifecycle.sql` — password reset и email verification schema changes
- `docs/README.md` — как запускать и вручную проверять сервис
- `docs/openapi.yaml` — актуальный OpenAPI/Swagger конфиг

## Следующий шаг

- стабилизировать публичный контракт reset/verify flow
- при необходимости разделить request/confirm endpoint'ы явнее
- усилить защиту от enumeration и rate limiting

## Текущие ограничения

- reset/verification токены больше не возвращаются в API-ответах
- доставка email зависит от доступности `notification-service`
- нет rate limiting и защиты от enumeration
- нет logout / revoke endpoint

## Smoke tests

Смоки лежат в `internal/http/router_smoke_test.go`.

Интеграционные тесты работы с PostgreSQL лежат в `internal/service/auth_integration_test.go`.

Swagger / service docs лежат в `docs/`.

Проверка:

```bash
make auth-test
```
