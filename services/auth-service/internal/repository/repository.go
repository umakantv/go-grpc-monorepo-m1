package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTokenNotFound = errors.New("token not found")
)

// Token represents a stored token (for refresh tokens and revocation)
type Token struct {
	ID        string
	UserID    string
	Token     string
	Type      string // "refresh" or "access"
	Revoked   bool
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Repository handles auth data access
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new auth repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// StoreToken stores a token in the database
func (r *Repository) StoreToken(ctx context.Context, token *Token) error {
	query := `
		INSERT INTO tokens (id, user_id, token, type, revoked, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.Token, token.Type, token.Revoked, token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// GetToken retrieves a token by its value
func (r *Repository) GetToken(ctx context.Context, tokenString string) (*Token, error) {
	query := `
		SELECT id, user_id, token, type, revoked, expires_at, created_at
		FROM tokens
		WHERE token = $1
	`

	var token Token
	err := r.db.QueryRowContext(ctx, query, tokenString).Scan(
		&token.ID, &token.UserID, &token.Token, &token.Type, &token.Revoked, &token.ExpiresAt, &token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return &token, nil
}

// RevokeToken revokes a token
func (r *Repository) RevokeToken(ctx context.Context, tokenString string) error {
	query := `UPDATE tokens SET revoked = true WHERE token = $1`

	result, err := r.db.ExecContext(ctx, query, tokenString)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrTokenNotFound
	}

	return nil
}

// RevokeAllUserTokens revokes all tokens for a user
func (r *Repository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	query := `UPDATE tokens SET revoked = true WHERE user_id = $1`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired tokens from the database
func (r *Repository) CleanupExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM tokens WHERE expires_at < CURRENT_TIMESTAMP`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	return nil
}
