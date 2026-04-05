# Архитектура

## api-gateway
Единая точка входа во внешний API. Содержит route groups (`/api/auth`, `/api/users`, `/api/items`, `/api/notifications`), middleware логирования и JWT-заглушку. В текущем состоянии остается placeholder-слоем без реального reverse proxy / service client flow и без БД.

## auth-service
Сервис аутентификации и account flow: регистрация, логин, refresh, сброс пароля, подтверждение email. Внутри владеет пользователями и одноразовыми токенами, а для email delivery вызывает `notification-service` по внутреннему HTTP.

## user-asset-service
Сервис пользовательского состояния: cloud watchlist, профиль и preferences. Работает поверх PostgreSQL, а для watchlist связывается с `catalog-service`, чтобы валидировать item ids и enrich'ить ответы базовой item metadata.

## api-integration-service
Слой runtime-интеграции с внешними торговыми площадками через адаптеры (`warframe`, `eve`, `tarkov`). Работает без собственной БД, выбирает provider по `game`, нормализует item data и pricing data в единый DTO. Для `tarkov` дополнительно учитывает `game_mode=regular|pve`. Для `eve` поиск должен оставаться в локальном `catalog-service`, а этот сервис отвечает за внешнюю item card и pricing fetch.

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

Дополнительно уже есть рабочий runtime flow для внешних pricing/item данных:

1. клиент или соседний сервис вызывает `api-integration-service`
2. `api-integration-service` выбирает provider по `game`
3. provider ходит во внешний API (`warframe.market`, `ESI`, `tarkov.dev`)
4. ответ приводится к общему item/pricing контракту
5. для `tarkov` один и тот же `item id` может давать разные цены в `regular` и `pve`

Дополнительно уже есть рабочий user-state flow:

1. клиент создает или обновляет профиль в `user-asset-service`
2. клиент добавляет предмет в watchlist по локальному `catalog item id`
3. `user-asset-service` валидирует item через `catalog-service`
4. `user-asset-service` хранит watchlist и preferences в PostgreSQL
5. при чтении watchlist сервис обогащает ответ item summary из каталога
