# Архитектура

## api-gateway
Единая точка входа во внешний API. Содержит route groups (`/api/auth`, `/api/users`, `/api/items`, `/api/notifications`), middleware логирования и JWT-заглушку. В текущем состоянии остается placeholder-слоем без реального reverse proxy / service client flow и без БД.

## auth-service
Сервис аутентификации и account flow: регистрация, логин, refresh, сброс пароля, подтверждение email. Внутри владеет пользователями и одноразовыми токенами, а для email delivery вызывает `notification-service` по внутреннему HTTP.

## user-asset-service
Сервис пользовательских активов: watchlist, недавние данные, предпочтения. Имеет рабочий CRUD-контур поверх PostgreSQL.

## api-integration-service
Слой интеграции с внешними торговыми площадками через адаптеры (`warframe`, `eve`, `tarkov`). В текущем состоянии в основном остается placeholder и работает без БД.

## catalog-service
Канонический каталог предметов: CRUD, поиск, локализации, ingestion через `POST /items/upsert`. Работает поверх PostgreSQL и уже используется как локальный source of truth для item metadata.

## notification-service
Сервис уведомлений с PostgreSQL outbox и абстракцией email-провайдера. Поддерживает `mock` provider и рабочий `Resend` provider, используется `auth-service` для password reset и email verification flow.

## catalog-importer
Отдельная CLI-утилита импорта данных каталога из источников (`warframe`, `eve`, `tarkov`). Работает как внешний ingestion client для `catalog-service` и потоково пишет metadata и локализации в локальный каталог.

## Хранилища
Отдельные БД PostgreSQL подключены только там, где на текущем этапе хранится состояние: auth, user-asset, catalog, notification.

Middleware:
- request_id
- logging
- auth

## Текущий рабочий вертикальный срез

Наиболее завершенный пользовательский сценарий сейчас проходит так:

1. клиент вызывает `auth-service`
2. `auth-service` создает/проверяет токены и состояние пользователя
3. при необходимости `auth-service` вызывает `notification-service`
4. `notification-service` пишет запись в `notification_outbox` и отправляет email через provider

Дополнительно уже есть рабочий data flow для каталога:

1. `catalog-importer` забирает metadata из `warframe`, `eve` или `tarkov`
2. `catalog-importer` пишет предметы в `catalog-service` через `POST /items/upsert`
3. `catalog-service` хранит базовые поля в `items`
4. `catalog-service` хранит локализации в `item_translations`
5. поиск предметов выполняется локально в PostgreSQL через `GET /items/search`
