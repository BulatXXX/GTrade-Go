# Архитектура

## api-gateway
Единая точка входа во внешний API. Содержит route groups (`/api/auth`, `/api/users`, `/api/items`, `/api/notifications`), middleware логирования и JWT-заглушку. В текущем skeleton работает без БД.

## auth-service
Сервис аутентификации и account flow: регистрация, логин, refresh, сброс пароля, подтверждение email.

## user-asset-service
Сервис пользовательских активов: watchlist, недавние данные, предпочтения.

## api-integration-service
Слой интеграции с внешними торговыми площадками через адаптеры (`warframe`, `eve`, `tarkov`). В текущем skeleton работает без БД.

## catalog-service
Каталог предметов и цен: чтение, поиск, upsert.

## notification-service
Сервис уведомлений с абстракцией email-провайдера. Подготовлены mock-провайдер и заготовка под Resend.

## catalog-importer
Отдельная CLI-утилита импорта данных каталога из источников (`warframe`, `eve`, `tarkov`).

## Хранилища
Отдельные БД PostgreSQL подключены только там, где в skeleton уже есть данные: auth, user-asset, catalog, notification.

Middleware:
- request_id
- logging
- auth