package auth

import (
    "database/sql"
    "errors"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"
)

const CookieName = "session_token"

func ttlFromEnv() time.Duration {
    if v := os.Getenv("SESSION_TTL"); v != "" {
        if d, err := time.ParseDuration(v); err == nil && d > 0 { return d }
    }
    return 30 * 24 * time.Hour
}

// cookieSecure determines the Secure flag for cookies. Defaults true in non-local.
func cookieSecure() bool {
    if v := strings.ToLower(os.Getenv("COOKIE_SECURE")); v != "" {
        return v == "1" || v == "true" || v == "yes"
    }
    // Default to secure unless explicitly disabled
    return true
}

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
    repo := NewRepository(db)
    api := r.Group("/api/auth")

    api.POST("/register", func(c *gin.Context) {
        var req struct{
            Email    string `json:"email"`
            Password string `json:"password"`
        }
        if err := c.BindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"}); return
        }
        req.Email = strings.TrimSpace(strings.ToLower(req.Email))
        if req.Email == "" || !strings.Contains(req.Email, "@") { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid email"}); return }
        if len(req.Password) < 12 { c.JSON(http.StatusBadRequest, gin.H{"error":"password too short (min 12)"}); return }

        // Hash password with bcrypt default cost
        hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"hash failed"}); return }

        // Create user
        u, err := repo.CreateUser(c.Request.Context(), req.Email, string(hash))
        if err != nil {
            if strings.Contains(strings.ToLower(err.Error()), "unique") {
                c.JSON(http.StatusConflict, gin.H{"error":"email already in use"}); return
            }
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
        }
        c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email})
    })

    api.POST("/login", func(c *gin.Context) {
        var req struct{
            Email    string `json:"email"`
            Password string `json:"password"`
        }
        if err := c.BindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"invalid json"}); return }
        req.Email = strings.TrimSpace(strings.ToLower(req.Email))
        if req.Email == "" || req.Password == "" { c.JSON(http.StatusBadRequest, gin.H{"error":"missing email or password"}); return }

        u, err := repo.GetUserByEmail(c.Request.Context(), req.Email)
        if err != nil { c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid credentials"}); return }
        if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error":"invalid credentials"}); return
        }

        s, err := repo.CreateSession(c.Request.Context(), u.ID, ttlFromEnv())
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error":"session failed"}); return }
        // Set secure, HTTP-only cookie
        maxAge := int(time.Until(s.ExpiresAt).Seconds())
        c.SetSameSite(http.SameSiteLaxMode)
        c.SetCookie(CookieName, s.Token, maxAge, "/", "", cookieSecure(), true)
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    api.POST("/logout", func(c *gin.Context) {
        tok, err := c.Cookie(CookieName)
        if err == nil && tok != "" { _ = repo.DeleteSession(c.Request.Context(), tok) }
        c.SetSameSite(http.SameSiteLaxMode)
        // overwrite with expired cookie
        c.SetCookie(CookieName, "", -1, "/", "", cookieSecure(), true)
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    api.GET("/me", func(c *gin.Context) {
        u, ok := CurrentUser(c, repo)
        if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
        c.JSON(http.StatusOK, gin.H{"id": u.ID, "email": u.Email})
    })
}

// CurrentUser resolves user from the session cookie for convenience.
func CurrentUser(c *gin.Context, repo *Repository) (User, bool) {
    tok, err := c.Cookie(CookieName)
    if err != nil || tok == "" { return User{}, false }
    u, err := repo.GetUserBySession(c.Request.Context(), tok)
    if err != nil { return User{}, false }
    return u, true
}

// AuthRequired middleware example (unused for now)
func AuthRequired(repo *Repository) gin.HandlerFunc {
    return func(c *gin.Context) {
        tok, err := c.Cookie(CookieName)
        if err != nil || tok == "" { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
        if _, err := repo.GetUserBySession(c.Request.Context(), tok); err != nil {
            if errors.Is(err, sql.ErrNoRows) { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"}); return }
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error":"auth failed"}); return
        }
        c.Next()
    }
}

