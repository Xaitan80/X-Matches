package auth

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "strings"
    "testing"
    "time"
    "strconv"

    "github.com/gin-gonic/gin"
    _ "modernc.org/sqlite"

    dbpkg "github.com/xaitan80/X-Matches/internal/db"
)

func newTestDB(t *testing.T) *sql.DB {
    t.Helper()
    dir := t.TempDir()
    dsn := filepath.Join(dir, "test.db")
    db, err := sql.Open("sqlite", dsn)
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    t.Cleanup(func(){ db.Close() })
    if err := dbpkg.Migrate(db); err != nil {
        t.Fatalf("migrate: %v", err)
    }
    return db
}

func newRouterWithAuth(t *testing.T, db *sql.DB) *gin.Engine {
    t.Helper()
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.Use(gin.Recovery())
    RegisterRoutes(r, db)
    // also mount admin routes for admin-related tests
    RegisterAdminRoutes(r, NewRepository(db))
    return r
}

func doJSON(r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
    var buf bytes.Buffer
    if body != nil {
        _ = json.NewEncoder(&buf).Encode(body)
    }
    req := httptest.NewRequest(method, path, &buf)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    return w
}

func doJSONWithCookie(r http.Handler, method, path string, body any, cookie string) *httptest.ResponseRecorder {
    var buf bytes.Buffer
    if body != nil {
        _ = json.NewEncoder(&buf).Encode(body)
    }
    req := httptest.NewRequest(method, path, &buf)
    req.Header.Set("Content-Type", "application/json")
    if cookie != "" {
        req.Header.Set("Cookie", cookie)
    }
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    return w
}

func cookieFrom(w *httptest.ResponseRecorder) string {
    sc := w.Header().Get("Set-Cookie")
    if sc == "" { return "" }
    // Return just the cookie pair (before the first ';')
    if i := strings.Index(sc, ";"); i > 0 {
        return sc[:i]
    }
    return sc
}

func TestRegister_InvalidJSON(t *testing.T) {
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader("{"))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d", w.Code)
    }
}

func TestRegister_InvalidEmail(t *testing.T) {
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // empty email
    w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"", "password":"123456789012"})
    if w.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", w.Code) }
    // missing @
    w = doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"userexample.com", "password":"123456789012"})
    if w.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", w.Code) }
}

func TestRegister_ShortPassword(t *testing.T) {
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // 11 chars => reject
    w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"user@example.com", "password":"12345678901"})
    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400 for short password, got %d", w.Code)
    }
}

func TestRegister_NormalizeAndSuccess(t *testing.T) {
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // Lowercasing + trimming
    w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"  USER@Example.COM  ", "password":"123456789012"})
    if w.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
    }
    var out map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &out)
    if out["email"].(string) != "user@example.com" {
        t.Fatalf("expected normalized email, got %v", out["email"]) 
    }
}

func TestRegister_DuplicateEmail(t *testing.T) {
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    body := map[string]any{"email":"dupe@example.com", "password":"123456789012"}
    w := doJSON(r, http.MethodPost, "/api/auth/register", body)
    if w.Code != http.StatusCreated { t.Fatalf("first create expected 201, got %d", w.Code) }
    w = doJSON(r, http.MethodPost, "/api/auth/register", body)
    if w.Code != http.StatusConflict {
        t.Fatalf("expected 409 for duplicate, got %d", w.Code)
    }
}

func TestLogin_SetsCookie(t *testing.T) {
    t.Setenv("COOKIE_SECURE", "false") // allow over HTTP in tests
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // create user
    w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"login@example.com", "password":"123456789012"})
    if w.Code != http.StatusCreated { t.Fatalf("register failed: %d", w.Code) }
    // login
    w = doJSON(r, http.MethodPost, "/api/auth/login", map[string]any{"email":"login@example.com", "password":"123456789012"})
    if w.Code != http.StatusOK { t.Fatalf("login expected 200, got %d", w.Code) }
    if sc := w.Header().Get("Set-Cookie"); !strings.Contains(sc, CookieName+"=") {
        t.Fatalf("expected Set-Cookie with %s, got %q", CookieName, sc)
    }
}

func TestLogout_ClearsSession(t *testing.T) {
    t.Setenv("COOKIE_SECURE", "false")
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // register
    w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"logout@example.com", "password":"123456789012"})
    if w.Code != http.StatusCreated { t.Fatalf("register failed: %d", w.Code) }
    // login
    w = doJSON(r, http.MethodPost, "/api/auth/login", map[string]any{"email":"logout@example.com", "password":"123456789012"})
    if w.Code != http.StatusOK { t.Fatalf("login failed: %d", w.Code) }
    ck := cookieFrom(w)
    if ck == "" { t.Fatalf("missing cookie") }
    // me should work
    w = doJSONWithCookie(r, http.MethodGet, "/api/auth/me", nil, ck)
    if w.Code != http.StatusOK { t.Fatalf("me expected 200, got %d", w.Code) }
    // logout
    w = doJSONWithCookie(r, http.MethodPost, "/api/auth/logout", nil, ck)
    if w.Code != http.StatusOK { t.Fatalf("logout expected 200, got %d", w.Code) }
    // me should be unauthorized now (old cookie no longer valid)
    w = doJSONWithCookie(r, http.MethodGet, "/api/auth/me", nil, ck)
    if w.Code != http.StatusUnauthorized { t.Fatalf("me expected 401 after logout, got %d", w.Code) }
}

func TestSession_Expiry(t *testing.T) {
    t.Setenv("COOKIE_SECURE", "false")
    // SQLite CURRENT_TIMESTAMP has second precision; use 1s TTL and sleep >1s
    t.Setenv("SESSION_TTL", "1s")
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // register
    w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"exp@example.com", "password":"123456789012"})
    if w.Code != http.StatusCreated { t.Fatalf("register failed: %d", w.Code) }
    // login
    w = doJSON(r, http.MethodPost, "/api/auth/login", map[string]any{"email":"exp@example.com", "password":"123456789012"})
    if w.Code != http.StatusOK { t.Fatalf("login failed: %d", w.Code) }
    ck := cookieFrom(w)
    if ck == "" { t.Fatalf("missing cookie") }
    // me initially OK
    w = doJSONWithCookie(r, http.MethodGet, "/api/auth/me", nil, ck)
    if w.Code != http.StatusOK { t.Fatalf("me expected 200, got %d", w.Code) }
    // wait for expiry (sleep > 1s)
    time.Sleep(2 * time.Second)
    // me should be 401 after expiry
    w = doJSONWithCookie(r, http.MethodGet, "/api/auth/me", nil, ck)
    if w.Code != http.StatusUnauthorized { t.Fatalf("me expected 401 after expiry, got %d", w.Code) }
}

func TestAuthRequired_Middleware(t *testing.T) {
    t.Setenv("COOKIE_SECURE", "false")
    db := newTestDB(t)
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.Use(gin.Recovery())
    repo := NewRepository(db)
    r.GET("/protected", AuthRequired(repo), func(c *gin.Context){ c.JSON(http.StatusOK, gin.H{"ok":true}) })
    // no cookie -> 401
    req := httptest.NewRequest(http.MethodGet, "/protected", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusUnauthorized { t.Fatalf("expected 401, got %d", w.Code) }
    // register+login
    rr := newRouterWithAuth(t, db)
    _ = doJSON(rr, http.MethodPost, "/api/auth/register", map[string]any{"email":"mw@example.com", "password":"123456789012"})
    lw := doJSON(rr, http.MethodPost, "/api/auth/login", map[string]any{"email":"mw@example.com", "password":"123456789012"})
    ck := cookieFrom(lw)
    // with cookie -> 200
    req = httptest.NewRequest(http.MethodGet, "/protected", nil)
    req.Header.Set("Cookie", ck)
    w = httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("expected 200 with cookie, got %d", w.Code) }
}

func loginAndGetCookie(t *testing.T, r http.Handler, email, password string) string {
    t.Helper()
    w := doJSON(r, http.MethodPost, "/api/auth/login", map[string]any{"email": email, "password": password})
    if w.Code != http.StatusOK {
        t.Fatalf("login failed for %s: %d", email, w.Code)
    }
    ck := cookieFrom(w)
    if ck == "" { t.Fatalf("missing cookie") }
    return ck
}

func TestAdmin_ListUsers_Gating(t *testing.T) {
    t.Setenv("COOKIE_SECURE", "false")
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // create a normal user and login
    _ = doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"user1@example.com", "password":"123456789012"})
    ckUser := loginAndGetCookie(t, r, "user1@example.com", "123456789012")
    // normal user should get 403
    w := doJSONWithCookie(r, http.MethodGet, "/api/admin/users", nil, ckUser)
    if w.Code != http.StatusForbidden { t.Fatalf("expected 403 for non-admin, got %d", w.Code) }

    // make admin via env list
    t.Setenv("ADMIN_EMAILS", "admin@example.com")
    _ = doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"admin@example.com", "password":"123456789012"})
    ckAdmin := loginAndGetCookie(t, r, "admin@example.com", "123456789012")
    w = doJSONWithCookie(r, http.MethodGet, "/api/admin/users", nil, ckAdmin)
    if w.Code != http.StatusOK { t.Fatalf("expected 200 for admin, got %d", w.Code) }
}

func TestAdmin_ResetPassword_Flow(t *testing.T) {
    t.Setenv("COOKIE_SECURE", "false")
    t.Setenv("ADMIN_EMAILS", "root@example.com")
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // create target user
    w := doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"target@example.com", "password":"oldpassword123"})
    if w.Code != http.StatusCreated { t.Fatalf("register target failed: %d", w.Code) }
    // create admin and login
    _ = doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"root@example.com", "password":"supersecurepass"})
    ckAdmin := loginAndGetCookie(t, r, "root@example.com", "supersecurepass")
    // find target id via admin list
    w = doJSONWithCookie(r, http.MethodGet, "/api/admin/users", nil, ckAdmin)
    if w.Code != http.StatusOK { t.Fatalf("list users failed: %d", w.Code) }
    var users []map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &users)
    var targetID int64
    for _, u := range users {
        if u["email"].(string) == "target@example.com" {
            // JSON numbers decode to float64
            targetID = int64(u["id"].(float64))
        }
    }
    if targetID == 0 { t.Fatalf("did not find target user id") }
    // reset password
    w = doJSONWithCookie(r, http.MethodPost, "/api/admin/users/"+strconv.FormatInt(targetID,10)+"/reset_password", map[string]any{"password":"newpassword456"}, ckAdmin)
    if w.Code != http.StatusOK { t.Fatalf("reset password failed: %d", w.Code) }
    // old password should fail
    w = doJSON(r, http.MethodPost, "/api/auth/login", map[string]any{"email":"target@example.com", "password":"oldpassword123"})
    if w.Code != http.StatusUnauthorized { t.Fatalf("old password should be invalid, got %d", w.Code) }
    // new password works
    w = doJSON(r, http.MethodPost, "/api/auth/login", map[string]any{"email":"target@example.com", "password":"newpassword456"})
    if w.Code != http.StatusOK { t.Fatalf("new password should work, got %d", w.Code) }
}

func TestAdmin_SetAdminFlag_Flow(t *testing.T) {
    t.Setenv("COOKIE_SECURE", "false")
    t.Setenv("ADMIN_EMAILS", "root@example.com")
    db := newTestDB(t)
    r := newRouterWithAuth(t, db)
    // create normal user and admin
    _ = doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"user2@example.com", "password":"strongpass123"})
    _ = doJSON(r, http.MethodPost, "/api/auth/register", map[string]any{"email":"root@example.com", "password":"supersecurepass"})
    ckAdmin := loginAndGetCookie(t, r, "root@example.com", "supersecurepass")

    // normal user cannot access admin
    ckUser := loginAndGetCookie(t, r, "user2@example.com", "strongpass123")
    w := doJSONWithCookie(r, http.MethodGet, "/api/admin/users", nil, ckUser)
    if w.Code != http.StatusForbidden { t.Fatalf("expected 403 before flag, got %d", w.Code) }

    // find user2 id
    w = doJSONWithCookie(r, http.MethodGet, "/api/admin/users", nil, ckAdmin)
    var users []map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &users)
    var targetID int64
    for _, u := range users { if u["email"].(string) == "user2@example.com" { targetID = int64(u["id"].(float64)) } }
    if targetID == 0 { t.Fatalf("no id for user2") }

    // set admin flag
    w = doJSONWithCookie(r, http.MethodPatch, "/api/admin/users/"+strconv.FormatInt(targetID,10)+"/admin", map[string]any{"is_admin": true}, ckAdmin)
    if w.Code != http.StatusOK { t.Fatalf("set admin failed: %d", w.Code) }

    // user2 should now be able to access admin
    w = doJSONWithCookie(r, http.MethodGet, "/api/admin/users", nil, ckUser)
    if w.Code != http.StatusOK { t.Fatalf("expected 200 after flag, got %d", w.Code) }
}
