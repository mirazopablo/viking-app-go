package apperrors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() {
	gin.SetMode(gin.TestMode)
}

func TestHandleErrorWithAppError(t *testing.T) {
	setupTestRouter()

	sentinel := errors.New("invalid credentials sentinel")
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		err := NewUnauthorized(CodeAuthInvalidCredentials, "Invalid email or password").WithInternal(sentinel)
		HandleError(c, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Success {
		t.Fatalf("expected success=false")
	}
	if body.Code != CodeAuthInvalidCredentials {
		t.Fatalf("expected code %s, got %s", CodeAuthInvalidCredentials, body.Code)
	}
	if body.Message != "Invalid email or password" {
		t.Fatalf("unexpected message: %s", body.Message)
	}
}

func TestHandleErrorHidesInternalDetailsOn500(t *testing.T) {
	setupTestRouter()

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		HandleError(c, errors.New("pq: password authentication failed for user postgres"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "postgres") {
		t.Fatalf("internal database error leaked in response body: %s", w.Body.String())
	}

	var body ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Code != CodeInternalError {
		t.Fatalf("expected code %s, got %s", CodeInternalError, body.Code)
	}
}

func TestTooManyRequestsHelper(t *testing.T) {
	setupTestRouter()

	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		TooManyRequests(c, CodeRateLimitExceeded, "Too many requests, please try again later")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", w.Code)
	}

	var body ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Code != CodeRateLimitExceeded {
		t.Fatalf("expected code %s, got %s", CodeRateLimitExceeded, body.Code)
	}
}

func TestOKAndCreatedHelpers(t *testing.T) {
	setupTestRouter()

	r := gin.New()
	r.GET("/ok", func(c *gin.Context) {
		OK(c, gin.H{"status": "ok"})
	})
	r.POST("/created", func(c *gin.Context) {
		Created(c, gin.H{"id": 1})
	})

	wOK := httptest.NewRecorder()
	r.ServeHTTP(wOK, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if wOK.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wOK.Code)
	}

	wCreated := httptest.NewRecorder()
	r.ServeHTTP(wCreated, httptest.NewRequest(http.MethodPost, "/created", nil))
	if wCreated.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", wCreated.Code)
	}
}

