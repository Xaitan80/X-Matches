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
}

// CreateUser inserts a new user. Returns conflict error if email exists.
func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
    res := User{}
    row := r.db.QueryRowContext(ctx,
        `INSERT INTO users (email, password_hash) VALUES (?, ?) RETURNING id, email, password_hash, created_at`,
        email, passwordHash,
    )
    if err := row.Scan(&res.ID, &res.Email, &res.PasswordHash, &res.CreatedAt); err != nil {
        return User{}, err
    }
    return res, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
    var u User
    err := r.db.QueryRowContext(ctx,
        `SELECT id, email, password_hash, created_at FROM users WHERE email = ?`, email,
    ).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
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
    if err != nil { return Session{}, err }
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
        SELECT u.id, u.email, u.password_hash, u.created_at
        FROM sessions s
        JOIN users u ON u.id = s.user_id
        WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP
    `, token).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
    if err != nil {
        return User{}, err
    }
    return u, nil
}

var ErrNotFound = errors.New("not found")

