package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	userv1 "github.com/yourorg/monorepo/gen/go/public/user"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// User represents a user in the database
type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Repository handles user data access
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new user repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetByID retrieves a user by ID
func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, name, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, name, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// Create creates a new user
func (r *Repository) Create(ctx context.Context, email, name, passwordHash string) (*User, error) {
	query := `
		INSERT INTO users (email, name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, name, created_at, updated_at
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, email, name, passwordHash).Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (r *Repository) Update(ctx context.Context, id string, updates map[string]interface{}) (*User, error) {
	// Build dynamic update query
	query := `UPDATE users SET updated_at = CURRENT_TIMESTAMP`
	args := []interface{}{id}
	argIdx := 2

	if email, ok := updates["email"].(string); ok {
		query += fmt.Sprintf(", email = $%d", argIdx)
		args = append(args, email)
		argIdx++
	}
	if name, ok := updates["name"].(string); ok {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, name)
		argIdx++
	}

	query += ` WHERE id = $1 RETURNING id, email, name, created_at, updated_at`

	var user User
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

// Delete deletes a user
func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// List retrieves users with pagination
func (r *Repository) List(ctx context.Context, pageSize int32, pageToken string) ([]*User, string, error) {
	query := `
		SELECT id, email, name, created_at, updated_at
		FROM users
		WHERE ($1 = '' OR id > $1)
		ORDER BY id
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, pageToken, pageSize+1)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating users: %w", err)
	}

	var nextPageToken string
	if len(users) > int(pageSize) {
		nextPageToken = users[pageSize-1].ID
		users = users[:pageSize]
	}

	return users, nextPageToken, nil
}

// ToProto converts a User to its proto representation
func (u *User) ToProto() *userv1.User {
	return &userv1.User{
		UserId: u.ID,
		Email:  u.Email,
		Name:   u.Name,
	}
}
