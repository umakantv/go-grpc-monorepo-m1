package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrFileNotFound = errors.New("file not found")
)

// FileStatus represents the lifecycle state of a file.
const (
	StatusPending  = "pending"
	StatusUploaded = "uploaded"
	StatusDeleted  = "deleted"
)

// File represents a row in the files table.
type File struct {
	ID            string
	Filename      string
	Mimetype      string
	Location      string // logical path prefix in S3
	SizeKB        int64
	OwnerEntity   string
	OwnerEntityID string
	Status        string
	S3Key         string
	URL           string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

// Repository handles file metadata persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new file repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new file record.
func (r *Repository) Create(ctx context.Context, f *File) (*File, error) {
	query := `
		INSERT INTO files (id, filename, mimetype, location, size_kb, owner_entity, owner_entity_id, status, s3_key, url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, filename, mimetype, location, size_kb, owner_entity, owner_entity_id, status, s3_key, url, created_at, updated_at
	`

	var created File
	err := r.db.QueryRowContext(ctx, query,
		f.ID, f.Filename, f.Mimetype, f.Location, f.SizeKB,
		f.OwnerEntity, f.OwnerEntityID, f.Status, f.S3Key, f.URL,
	).Scan(
		&created.ID, &created.Filename, &created.Mimetype, &created.Location,
		&created.SizeKB, &created.OwnerEntity, &created.OwnerEntityID,
		&created.Status, &created.S3Key, &created.URL,
		&created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	return &created, nil
}

// GetByID retrieves a file by ID (excluding soft-deleted).
func (r *Repository) GetByID(ctx context.Context, id string) (*File, error) {
	query := `
		SELECT id, filename, mimetype, location, size_kb, owner_entity, owner_entity_id,
		       status, s3_key, url, created_at, updated_at, deleted_at
		FROM files
		WHERE id = $1 AND deleted_at IS NULL
	`

	var f File
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID, &f.Filename, &f.Mimetype, &f.Location,
		&f.SizeKB, &f.OwnerEntity, &f.OwnerEntityID,
		&f.Status, &f.S3Key, &f.URL,
		&f.CreatedAt, &f.UpdatedAt, &f.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return &f, nil
}

// ConfirmUpload sets the file status to "uploaded".
func (r *Repository) ConfirmUpload(ctx context.Context, id string) (*File, error) {
	query := `
		UPDATE files SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING id, filename, mimetype, location, size_kb, owner_entity, owner_entity_id,
		          status, s3_key, url, created_at, updated_at
	`

	var f File
	err := r.db.QueryRowContext(ctx, query, StatusUploaded, id).Scan(
		&f.ID, &f.Filename, &f.Mimetype, &f.Location,
		&f.SizeKB, &f.OwnerEntity, &f.OwnerEntityID,
		&f.Status, &f.S3Key, &f.URL,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to confirm upload: %w", err)
	}
	return &f, nil
}

// SoftDelete marks a file as deleted.
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE files SET status = $1, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, StatusDeleted, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete file: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return ErrFileNotFound
	}
	return nil
}

// List retrieves files with optional owner filtering and pagination.
func (r *Repository) List(ctx context.Context, ownerEntity, ownerEntityID string, pageSize int32, pageToken string) ([]*File, string, error) {
	query := `
		SELECT id, filename, mimetype, location, size_kb, owner_entity, owner_entity_id,
		       status, s3_key, url, created_at, updated_at
		FROM files
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR owner_entity = $1)
		  AND ($2 = '' OR owner_entity_id = $2)
		  AND ($3 = '' OR id > $3)
		ORDER BY id
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, ownerEntity, ownerEntityID, pageToken, pageSize+1)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var f File
		if err := rows.Scan(
			&f.ID, &f.Filename, &f.Mimetype, &f.Location,
			&f.SizeKB, &f.OwnerEntity, &f.OwnerEntityID,
			&f.Status, &f.S3Key, &f.URL,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, &f)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating files: %w", err)
	}

	var nextPageToken string
	if len(files) > int(pageSize) {
		nextPageToken = files[pageSize-1].ID
		files = files[:pageSize]
	}

	return files, nextPageToken, nil
}