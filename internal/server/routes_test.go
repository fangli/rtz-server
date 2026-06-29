package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/gin-gonic/gin"
)

func TestPreservedRoutesAlwaysReturnsEmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1(router.Group("/v1"), &config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/devices/51f15e1f00000001/routes/preserved", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusOK, resp.Code, resp.Body.String())
	}
	if got := resp.Body.String(); got != "[]" {
		t.Fatalf("expected body [] without trailing newline, got %q", got)
	}
	if got := resp.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
}
