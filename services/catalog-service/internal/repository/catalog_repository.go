package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"gtrade/services/catalog-service/internal/model"
)

var (
	ErrNotImplemented = errors.New("catalog repository not implemented")
	ErrItemExists     = errors.New("item already exists")
	ErrItemNotFound   = errors.New("item not found")
)

type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: pool}
}

func (r *CatalogRepository) CreateItem(ctx context.Context, input model.CreateItemInput) (*model.Item, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer rollback(ctx, tx)

	id, err := newItemID()
	if err != nil {
		return nil, fmt.Errorf("generate item id: %w", err)
	}

	var item model.Item
	query := `
		INSERT INTO items (id, game, source, external_id, slug, name, description, image_url, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE)
		RETURNING id, game, source, external_id, slug, name, description, image_url, is_active, created_at, updated_at
	`
	if err := scanItem(tx.QueryRow(
		ctx,
		query,
		id,
		input.Game,
		input.Source,
		input.ExternalID,
		input.Slug,
		input.Name,
		nullIfEmpty(input.Description),
		nullIfEmpty(input.ImageURL),
	), &item); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrItemExists
		}
		return nil, fmt.Errorf("insert item: %w", err)
	}

	if err := upsertTranslations(ctx, tx, item.ID, input.Translations, false); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create item: %w", err)
	}

	return r.GetItemByID(ctx, item.ID)
}

func (r *CatalogRepository) UpdateItem(ctx context.Context, id string, input model.UpdateItemInput) (*model.Item, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer rollback(ctx, tx)

	res, err := tx.Exec(
		ctx,
		`
		UPDATE items
		SET
			slug = CASE WHEN $2 = '' THEN slug ELSE $2 END,
			name = CASE WHEN $3 = '' THEN name ELSE $3 END,
			description = CASE WHEN $4 = '' THEN description ELSE $4 END,
			image_url = CASE WHEN $5 = '' THEN image_url ELSE $5 END,
			is_active = COALESCE($6, is_active),
			updated_at = NOW()
		WHERE id = $1
		`,
		id,
		input.Slug,
		input.Name,
		input.Description,
		input.ImageURL,
		input.IsActive,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrItemExists
		}
		return nil, fmt.Errorf("update item: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, ErrItemNotFound
	}

	if input.Translations != nil {
		if err := upsertTranslations(ctx, tx, id, input.Translations, true); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update item: %w", err)
	}

	return r.GetItemByID(ctx, id)
}

func (r *CatalogRepository) DeactivateItem(ctx context.Context, id string) error {
	res, err := r.pool.Exec(ctx, `UPDATE items SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate item: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *CatalogRepository) GetItemByID(ctx context.Context, id string) (*model.Item, error) {
	item, err := r.getItem(ctx, r.pool, `SELECT id, game, source, external_id, slug, name, description, image_url, is_active, created_at, updated_at FROM items WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *CatalogRepository) ListItems(ctx context.Context, filter model.ListItemsFilter) ([]model.Item, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	args := []any{limit, filter.Offset}
	var clauses []string
	argPos := 3
	if strings.TrimSpace(filter.Game) != "" {
		clauses = append(clauses, fmt.Sprintf("game = $%d", argPos))
		args = append(args, strings.TrimSpace(filter.Game))
		argPos++
	}
	if strings.TrimSpace(filter.Source) != "" {
		clauses = append(clauses, fmt.Sprintf("source = $%d", argPos))
		args = append(args, strings.TrimSpace(filter.Source))
		argPos++
	}
	if filter.ActiveOnly != nil {
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *filter.ActiveOnly)
		argPos++
	}

	query := `
		SELECT id, game, source, external_id, slug, name, description, image_url, is_active, created_at, updated_at
		FROM items
	`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var item model.Item
		if err := scanItemRow(rows, &item); err != nil {
			return nil, fmt.Errorf("scan item row: %w", err)
		}

		translations, err := r.listTranslations(ctx, r.pool, item.ID)
		if err != nil {
			return nil, err
		}
		item.Translations = translations
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("item rows: %w", err)
	}

	return items, nil
}

func (r *CatalogRepository) SearchItems(ctx context.Context, filter model.SearchItemsFilter) ([]model.Item, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	activeOnly := true
	if filter.ActiveOnly != nil {
		activeOnly = *filter.ActiveOnly
	}

	args := []any{"%" + strings.TrimSpace(filter.Query) + "%", activeOnly}
	conditions := []string{"(i.name ILIKE $1", "i.is_active = $2"}
	argPos := 3

	if strings.TrimSpace(filter.Language) != "" {
		conditions[0] += fmt.Sprintf(" OR (t.language_code = $%d AND t.name ILIKE $1)", argPos)
		args = append(args, strings.TrimSpace(filter.Language))
		argPos++
	}
	conditions[0] += ")"

	if strings.TrimSpace(filter.Game) != "" {
		conditions = append(conditions, fmt.Sprintf("i.game = $%d", argPos))
		args = append(args, strings.TrimSpace(filter.Game))
		argPos++
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT i.id, i.game, i.source, i.external_id, i.slug, i.name, i.description, i.image_url, i.is_active, i.created_at, i.updated_at
		FROM items i
		LEFT JOIN item_translations t ON t.item_id = i.id
		WHERE %s
		ORDER BY i.created_at DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), argPos, argPos+1)
	args = append(args, limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search items: %w", err)
	}
	defer rows.Close()

	var items []model.Item
	for rows.Next() {
		var item model.Item
		if err := scanItemRow(rows, &item); err != nil {
			return nil, fmt.Errorf("scan search item row: %w", err)
		}
		translations, err := r.listTranslations(ctx, r.pool, item.ID)
		if err != nil {
			return nil, err
		}
		item.Translations = translations
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search item rows: %w", err)
	}

	return items, nil
}

func (r *CatalogRepository) getItem(ctx context.Context, q queryRower, query string, args ...any) (*model.Item, error) {
	var item model.Item
	if err := scanItem(q.QueryRow(ctx, query, args...), &item); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("get item by id: %w", err)
	}

	translations, err := r.listTranslations(ctx, q, item.ID)
	if err != nil {
		return nil, err
	}
	item.Translations = translations
	return &item, nil
}

func (r *CatalogRepository) listTranslations(ctx context.Context, q queryRower, itemID string) ([]model.ItemTranslation, error) {
	rows, err := q.Query(ctx, `
		SELECT item_id, language_code, name, description
		FROM item_translations
		WHERE item_id = $1
		ORDER BY language_code ASC
	`, itemID)
	if err != nil {
		return nil, fmt.Errorf("query translations: %w", err)
	}
	defer rows.Close()

	var translations []model.ItemTranslation
	for rows.Next() {
		var translation model.ItemTranslation
		var description *string
		if err := rows.Scan(&translation.ItemID, &translation.LanguageCode, &translation.Name, &description); err != nil {
			return nil, fmt.Errorf("scan translation row: %w", err)
		}
		translation.Description = stringValue(description)
		translations = append(translations, translation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("translation rows: %w", err)
	}
	return translations, nil
}

func upsertTranslations(ctx context.Context, tx pgx.Tx, itemID string, translations []model.ItemTranslation, replace bool) error {
	if replace {
		if _, err := tx.Exec(ctx, `DELETE FROM item_translations WHERE item_id = $1`, itemID); err != nil {
			return fmt.Errorf("delete translations: %w", err)
		}
	}

	for _, translation := range translations {
		if _, err := tx.Exec(
			ctx,
			`
			INSERT INTO item_translations (item_id, language_code, name, description)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (item_id, language_code)
			DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description
			`,
			itemID,
			translation.LanguageCode,
			translation.Name,
			nullIfEmpty(translation.Description),
		); err != nil {
			return fmt.Errorf("upsert translation: %w", err)
		}
	}

	return nil
}

func newItemID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "item_" + hex.EncodeToString(raw[:]), nil
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func scanItem(row pgx.Row, item *model.Item) error {
	var description *string
	var imageURL *string
	if err := row.Scan(
		&item.ID,
		&item.Game,
		&item.Source,
		&item.ExternalID,
		&item.Slug,
		&item.Name,
		&description,
		&imageURL,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return err
	}
	item.Description = stringValue(description)
	item.ImageURL = stringValue(imageURL)
	return nil
}

func scanItemRow(rows pgx.Rows, item *model.Item) error {
	var description *string
	var imageURL *string
	if err := rows.Scan(
		&item.ID,
		&item.Game,
		&item.Source,
		&item.ExternalID,
		&item.Slug,
		&item.Name,
		&description,
		&imageURL,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return err
	}
	item.Description = stringValue(description)
	item.ImageURL = stringValue(imageURL)
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}
