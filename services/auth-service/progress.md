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

- `POST /password/reset/request` создает reset token, сохраняет его в БД и сейчас возвращает его в API-ответе
- `POST /password/reset/confirm` принимает reset token и новый пароль, затем меняет пароль
- `POST /email/verify` в режиме request создает verification token и сейчас возвращает его в API-ответе
- `POST /email/verify` в режиме confirm принимает verification token и помечает email как подтвержденный

Такой flow подходит для локальной разработки и ручной проверки, но не является production-ready.

## Что нужно доделать

- интеграция `auth-service` с `notification-service`
- отправка reset token и verification token через email, а не в API-ответе
- разделение request/confirm сценариев на более явный публичный контракт при необходимости
- валидация email и password на уровне входных DTO
- защита от user enumeration в reset/verify request flow
- rate limiting / throttling для чувствительных auth endpoint'ов
- logout / revoke flow
- account endpoint'ы вроде `GET /me` при необходимости
- swagger-ui или другой способ локально просматривать OpenAPI

## Smoke tests

Смоки лежат в `internal/http/router_smoke_test.go`.

Интеграционные тесты работы с PostgreSQL лежат в `internal/service/auth_integration_test.go`.

Swagger / service docs лежат в `docs/`.

Проверка:

```bash
make auth-test
```
