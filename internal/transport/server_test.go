package transport

import (
	"archive-release/internal/application"
	"archive-release/internal/persistence"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrowserWorkbenchIncludesReleaseWorkflow(t *testing.T) {
	server := New(application.New(persistence.NewMemory(), "test-release-key"))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("首页状态码 = %d，期望 %d", response.Code, http.StatusOK)
	}
	for _, marker := range []string{"case-form", "revision-form", "findings-list", "review-form", "freeze-button", "verify-button"} {
		if !strings.Contains(response.Body.String(), marker) {
			t.Errorf("首页缺少工作流控件 %q", marker)
		}
	}
}

func TestBrowserAssetsAreServed(t *testing.T) {
	server := New(application.New(persistence.NewMemory(), "test-release-key"))
	for _, path := range []string{"/static/style.css", "/static/app.js"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("%s 状态码 = %d，期望 %d", path, response.Code, http.StatusOK)
		}
	}
}
