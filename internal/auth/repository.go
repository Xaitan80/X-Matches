package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

type Repository struct {
    db *sql.DB
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	IsAdmin      bool
}

// CreateUser inserts a new user. Returns conflict error if email exists.
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var cnt int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&cnt); err != nil {
		return User{}, err
	}
	isAdmin := 0
	if cnt == 0 { // first user becomes admin
		isAdmin = 1
	}
	row := tx.QueryRowContext(ctx,
		`INSERT INTO users (email, password_hash, is_admin) VALUES (?, ?, ?) RETURNING id, email, password_hash, created_at, COALESCE(is_admin,0)`,
		email, passwordHash, isAdmin,
	)
	var res User
	if err := row.Scan(&res.ID, &res.Email, &res.PasswordHash, &res.CreatedAt, &res.IsAdmin); err != nil {
		return User{}, err
	}
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return res, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at, COALESCE(is_admin,0) FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.IsAdmin)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, created_at, COALESCE(is_admin,0) FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.IsAdmin)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// NewToken returns a cryptographically secure random token (hex-64)
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *Repository) CreateSession(ctx context.Context, userID int64, ttl time.Duration) (Session, error) {
	tok, err := NewToken()
	if err != nil {
		return Session{}, err
	}
	exp := time.Now().Add(ttl).UTC()
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?) RETURNING token, user_id, expires_at, created_at`,
		tok, userID, exp,
	)
	var s Session
	if err := row.Scan(&s.Token, &s.UserID, &s.ExpiresAt, &s.CreatedAt); err != nil {
		return Session{}, err
	}
	return s, nil
}

func (r *Repository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (r *Repository) GetUserBySession(ctx context.Context, token string) (User, error) {
	var u User
	// Clean up expired while checking
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`); err != nil {
		// non-fatal
		_ = err
	}
	err := r.db.QueryRowContext(ctx, `
        SELECT u.id, u.email, u.password_hash, u.created_at, COALESCE(u.is_admin,0)
        FROM sessions s
        JOIN users u ON u.id = s.user_id
        WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP
    `, token).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.IsAdmin)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

var ErrNotFound = errors.New("not found")

// Admin utilities
func (r *Repository) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, email, password_hash, created_at, COALESCE(is_admin,0) FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.IsAdmin); err != nil {
			return nil, err
		}
		// don't expose password hash to callers casually; callers should omit it
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) SetPasswordHash(ctx context.Context, userID int64, newHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, newHash, userID)
	return err
}

func (r *Repository) SetAdmin(ctx context.Context, userID int64, isAdmin bool) error {
	val := 0
	if isAdmin {
		val = 1
	}
	_, err := r.db.ExecContext(ctx, `UPDATE users SET is_admin = ? WHERE id = ?`, val, userID)
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, userID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) CountOtherAdmins(ctx context.Context, excludeID int64) (int64, error) {
    var n int64
    err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE is_admin = 1 AND id != ?`, excludeID).Scan(&n)
    return n, err
}

func (r *Repository) UpdateEmail(ctx context.Context, userID int64, email string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE users SET email = ? WHERE id = ?`, email, userID)
    return err
}

// Email reservation helpers
func (r *Repository) IsEmailReserved(ctx context.Context, email string) (bool, *int64, error) {
    var reservedBy sql.NullInt64
    err := r.db.QueryRowContext(ctx, `SELECT reserved_by FROM used_emails WHERE email = ?`, email).Scan(&reservedBy)
    if err == sql.ErrNoRows {
        return false, nil, nil
    }
    if err != nil { return false, nil, err }
    if reservedBy.Valid {
        v := reservedBy.Int64
        return true, &v, nil
    }
    return true, nil, nil
}

func (r *Repository) ReserveEmail(ctx context.Context, email string, userID *int64) error {
    // Upsert-like: insert if not exists
    if userID != nil {
        _, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO used_emails(email, reserved_by) VALUES(?, ?)`, email, *userID)
        return err
    }
    _, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO used_emails(email, reserved_by) VALUES(?, NULL)`, email)
    return err
}
