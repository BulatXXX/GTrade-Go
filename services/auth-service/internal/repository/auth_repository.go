package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmailExists = errors.New("email already exists")

type User struct {
	ID            int64
	Email         string
	PasswordHash  string
	EmailVerified bool
	CreatedAt     time.Time
}

type RefreshToken struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type OneTimeToken struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	UsedAt    *time.Time
}

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	query := `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, email_verified, created_at
	`

	var user User
	if err := r.pool.QueryRow(ctx, query, email, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified, &user.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return &user, nil
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, email, password_hash, email_verified, created_at FROM users WHERE email = $1`

	var user User
	if err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified, &user.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	query := `SELECT id, email, password_hash, email_verified, created_at FROM users WHERE id = $1`

	var user User
	if err := r.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.EmailVerified, &user.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *AuthRepository) UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2 WHERE id = $1`
	if _, err := r.pool.Exec(ctx, query, userID, passwordHash); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	return nil
}

func (r *AuthRepository) MarkEmailVerified(ctx context.Context, userID int64) error {
	query := `UPDATE users SET email_verified = TRUE WHERE id = $1`
	if _, err := r.pool.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	return nil
}

func (r *AuthRepository) SaveRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	query := `INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, query, userID, token, expiresAt); err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token = $1
	`

	var rt RefreshToken
	if err := r.pool.QueryRow(ctx, query, token).Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &rt, nil
}

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token = $1 AND revoked_at IS NULL`
	if _, err := r.pool.Exec(ctx, query, token); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *AuthRepository) SavePasswordResetToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	query := `INSERT INTO password_reset_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, query, userID, token, expiresAt); err != nil {
		return fmt.Errorf("save password reset token: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetPasswordResetToken(ctx context.Context, token string) (*OneTimeToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, used_at
		FROM password_reset_tokens
		WHERE token = $1
	`

	return scanOneTimeToken(ctx, r.pool, query, token)
}

func (r *AuthRepository) UsePasswordResetToken(ctx context.Context, token string) error {
	query := `UPDATE password_reset_tokens SET used_at = NOW() WHERE token = $1 AND used_at IS NULL`
	if _, err := r.pool.Exec(ctx, query, token); err != nil {
		return fmt.Errorf("use password reset token: %w", err)
	}
	return nil
}

func (r *AuthRepository) SaveEmailVerificationToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	query := `INSERT INTO email_verification_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, query, userID, token, expiresAt); err != nil {
		return fmt.Errorf("save email verification token: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetEmailVerificationToken(ctx context.Context, token string) (*OneTimeToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, used_at
		FROM email_verification_tokens
		WHERE token = $1
	`

	return scanOneTimeToken(ctx, r.pool, query, token)
}

func (r *AuthRepository) UseEmailVerificationToken(ctx context.Context, token string) error {
	query := `UPDATE email_verification_tokens SET used_at = NOW() WHERE token = $1 AND used_at IS NULL`
	if _, err := r.pool.Exec(ctx, query, token); err != nil {
		return fmt.Errorf("use email verification token: %w", err)
	}
	return nil
}

func scanOneTimeToken(ctx context.Context, pool *pgxpool.Pool, query, token string) (*OneTimeToken, error) {
	var ot OneTimeToken
	if err := pool.QueryRow(ctx, query, token).Scan(&ot.ID, &ot.UserID, &ot.Token, &ot.ExpiresAt, &ot.CreatedAt, &ot.UsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get one-time token: %w", err)
	}
	return &ot, nil
}
