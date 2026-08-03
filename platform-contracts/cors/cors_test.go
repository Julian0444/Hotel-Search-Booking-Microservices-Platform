package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Middleware())
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return router
}

func doRequest(router *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/ping", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestAllowlistedOriginIsReflectedWithVary(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173, http://localhost:3000")
	router := newTestRouter()

	rec := doRequest(router, http.MethodGet, "http://localhost:5173")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want the allowlisted origin", got)
	}
	if got := rec.Header().Values("Vary"); len(got) == 0 {
		t.Error("expected Vary: Origin when reflecting an allowlisted origin")
	}
}

func TestDisallowedOriginGetsNoCORSHeaders(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	router := newTestRouter()

	rec := doRequest(router, http.MethodGet, "https://evil.example")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want empty for a disallowed origin", got)
	}
}

// C1/RV27: el wildcard responde el literal "*" (no refleja el Origin) y jamás
// se emite Access-Control-Allow-Credentials.
func TestWildcardSendsLiteralStarWithoutCredentials(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	router := newTestRouter()

	rec := doRequest(router, http.MethodGet, "https://anything.example")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want literal *", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials must never be sent, got %q", got)
	}
}

func TestPreflightRespondsNoContent(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	router := newTestRouter()

	rec := doRequest(router, http.MethodOptions, "http://localhost:5173")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("preflight must include Access-Control-Allow-Methods")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("preflight must include Access-Control-Allow-Headers")
	}
}

func TestDefaultAllowsViteDevServer(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	router := newTestRouter()

	rec := doRequest(router, http.MethodGet, "http://localhost:5173")

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("default allowlist should include the Vite dev server, got %q", got)
	}
}
