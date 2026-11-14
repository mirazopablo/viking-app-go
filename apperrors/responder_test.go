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

func TestValidateDateRejectsSQLiAndTruncatedPayloads(t *testing.T) {
	setupTestRouter()

	r := gin.New()
	r.GET("/avail", func(c *gin.Context) {
		date := c.Query("date")
		if !ValidateDate(c, date, "date") {
			return
		}
		OK(c, gin.H{"date": date})
	})

	cases := []struct {
		name       string
		query      string
		wantStatus int
		wantCode   string
	}{
		{"valid date", "?date=2026-11-20", http.StatusOK, ""},
		{"sqli basic", "?date='%20OR%201=1;--", http.StatusBadRequest, CodeValidationError},
		{"sqli aggressive with valid prefix", "?date=2026-11-20';%20SELECT%20pg_sleep(5);--", http.StatusBadRequest, CodeValidationError},
		{"invalid calendar date", "?date=2026-02-31", http.StatusBadRequest, CodeValidationError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/avail"+tc.query, nil))
			if w.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d (body: %s)", tc.wantStatus, w.Code, w.Body.String())
			}
			if tc.wantCode != "" {
				var body ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if body.Code != tc.wantCode {
					t.Fatalf("expected code %s, got %s", tc.wantCode, body.Code)
				}
			}
		})
	}
}

func TestValidateUUIDRejectsSQLi(t *testing.T) {
	setupTestRouter()

	r := gin.New()
	r.GET("/item/:id", func(c *gin.Context) {
		if !ValidateUUID(c, c.Param("id"), "id") {
			return
		}
		OK(c, gin.H{"id": c.Param("id")})
	})

	wBad := httptest.NewRecorder()
	r.ServeHTTP(wBad, httptest.NewRequest(http.MethodGet, "/item/not-a-uuid", nil))
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", wBad.Code)
	}

	wGood := httptest.NewRecorder()
	r.ServeHTTP(wGood, httptest.NewRequest(http.MethodGet, "/item/4d1e72f7-6d02-4774-a037-7a2311ed873f", nil))
	if wGood.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wGood.Code)
	}
}

