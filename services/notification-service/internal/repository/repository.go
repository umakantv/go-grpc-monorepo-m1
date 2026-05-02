package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrDeviceTokenNotFound  = errors.New("device token not found")
	ErrNotificationNotFound = errors.New("notification not found")
)

const (
	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"
)

// DeviceToken represents a row in the device_tokens table.
type DeviceToken struct {
	ID         string
	UserID     string
	Token      string
	Platform   string // "ios", "android", "web"
	DeviceName string
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Notification represents a row in the notifications table.
type Notification struct {
	ID            string
	UserID        string
	Title         string
	Body          string
	Data          map[string]string
	ImageURL      string
	Status        string
	FailureReason string
	SentAt        *time.Time
	CreatedAt     time.Time
}

// Repository handles notification data access.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new notification repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ---- Device token operations ----

// UpsertDeviceToken inserts a new device token or re-activates an existing one.
// If the token already exists (possibly for a different user), it is reassigned.
func (r *Repository) UpsertDeviceToken(ctx context.Context, dt *DeviceToken) (*DeviceToken, error) {
	query := `
		INSERT INTO device_tokens (id, user_id, token, platform, device_name, active)
		VALUES ($1, $2, $3, $4, $5, TRUE)
		ON CONFLICT (token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			platform = EXCLUDED.platform,
			device_name = EXCLUDED.device_name,
			active = TRUE,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, user_id, token, platform, device_name, active, created_at, updated_at
	`

	var out DeviceToken
	err := r.db.QueryRowContext(ctx, query,
		dt.ID, dt.UserID, dt.Token, dt.Platform, dt.DeviceName,
	).Scan(
		&out.ID, &out.UserID, &out.Token, &out.Platform,
		&out.DeviceName, &out.Active, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert device token: %w", err)
	}
	return &out, nil
}

// DeactivateDeviceToken marks a device token as inactive.
func (r *Repository) DeactivateDeviceToken(ctx context.Context, token string) error {
	query := `UPDATE device_tokens SET active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE token = $1`
	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to deactivate device token: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDeviceTokenNotFound
	}
	return nil
}

// DeactivateDeviceTokens marks multiple tokens as inactive in one query.
func (r *Repository) DeactivateDeviceTokens(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	// Build query with positional params: $1, $2, ...
	query := `UPDATE device_tokens SET active = FALSE, updated_at = CURRENT_TIMESTAMP WHERE token IN (`
	args := make([]interface{}, len(tokens))
	for i, t := range tokens {
		if i > 0 {
			query += ","
		}
		query += fmt.Sprintf("$%d", i+1)
		args[i] = t
	}
	query += ")"

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to deactivate device tokens: %w", err)
	}
	return nil
}

// ListActiveTokensByUserID returns all active device tokens for a user.
func (r *Repository) ListActiveTokensByUserID(ctx context.Context, userID string) ([]*DeviceToken, error) {
	query := `
		SELECT id, user_id, token, platform, device_name, active, created_at, updated_at
		FROM device_tokens
		WHERE user_id = $1 AND active = TRUE
		ORDER BY created_at DESC
	`
	return r.scanDeviceTokens(ctx, query, userID)
}

// ListTokensByUserID returns all device tokens for a user (active + inactive).
func (r *Repository) ListTokensByUserID(ctx context.Context, userID string) ([]*DeviceToken, error) {
	query := `
		SELECT id, user_id, token, platform, device_name, active, created_at, updated_at
		FROM device_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	return r.scanDeviceTokens(ctx, query, userID)
}

func (r *Repository) scanDeviceTokens(ctx context.Context, query string, args ...interface{}) ([]*DeviceToken, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*DeviceToken
	for rows.Next() {
		var dt DeviceToken
		if err := rows.Scan(
			&dt.ID, &dt.UserID, &dt.Token, &dt.Platform,
			&dt.DeviceName, &dt.Active, &dt.CreatedAt, &dt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device token: %w", err)
		}
		tokens = append(tokens, &dt)
	}
	return tokens, rows.Err()
}

// ---- Notification operations ----

// CreateNotification inserts a notification record.
func (r *Repository) CreateNotification(ctx context.Context, n *Notification) (*Notification, error) {
	dataJSON, _ := json.Marshal(n.Data)

	query := `
		INSERT INTO notifications (id, user_id, title, body, data, image_url, status, failure_reason, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, title, body, data, image_url, status, failure_reason, sent_at, created_at
	`

	var out Notification
	var rawData sql.NullString
	var failureReason sql.NullString
	var sentAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query,
		n.ID, n.UserID, n.Title, n.Body, string(dataJSON),
		n.ImageURL, n.Status, nullString(n.FailureReason), nullTime(n.SentAt),
	).Scan(
		&out.ID, &out.UserID, &out.Title, &out.Body, &rawData,
		&out.ImageURL, &out.Status, &failureReason, &sentAt, &out.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	if rawData.Valid {
		_ = json.Unmarshal([]byte(rawData.String), &out.Data)
	}
	out.FailureReason = failureReason.String
	if sentAt.Valid {
		out.SentAt = &sentAt.Time
	}

	return &out, nil
}

// GetNotification retrieves a notification by ID.
func (r *Repository) GetNotification(ctx context.Context, id string) (*Notification, error) {
	query := `
		SELECT id, user_id, title, body, data, image_url, status, failure_reason, sent_at, created_at
		FROM notifications
		WHERE id = $1
	`

	var n Notification
	var rawData sql.NullString
	var failureReason sql.NullString
	var sentAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&n.ID, &n.UserID, &n.Title, &n.Body, &rawData,
		&n.ImageURL, &n.Status, &failureReason, &sentAt, &n.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	if rawData.Valid {
		_ = json.Unmarshal([]byte(rawData.String), &n.Data)
	}
	n.FailureReason = failureReason.String
	if sentAt.Valid {
		n.SentAt = &sentAt.Time
	}

	return &n, nil
}

// ListNotifications retrieves notifications for a user with pagination.
func (r *Repository) ListNotifications(ctx context.Context, userID string, pageSize int32, pageToken string) ([]*Notification, string, error) {
	query := `
		SELECT id, user_id, title, body, data, image_url, status, failure_reason, sent_at, created_at
		FROM notifications
		WHERE user_id = $1 AND ($2 = '' OR id > $2)
		ORDER BY id
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, pageToken, pageSize+1)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var n Notification
		var rawData sql.NullString
		var failureReason sql.NullString
		var sentAt sql.NullTime

		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Title, &n.Body, &rawData,
			&n.ImageURL, &n.Status, &failureReason, &sentAt, &n.CreatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("failed to scan notification: %w", err)
		}

		if rawData.Valid {
			_ = json.Unmarshal([]byte(rawData.String), &n.Data)
		}
		n.FailureReason = failureReason.String
		if sentAt.Valid {
			n.SentAt = &sentAt.Time
		}
		notifications = append(notifications, &n)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating notifications: %w", err)
	}

	var nextPageToken string
	if len(notifications) > int(pageSize) {
		nextPageToken = notifications[pageSize-1].ID
		notifications = notifications[:pageSize]
	}

	return notifications, nextPageToken, nil
}

// helpers

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}