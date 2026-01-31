package httputil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

//nolint:gocognit
func TestParseuuidParam(t *testing.T) {
	tests := []struct {
		name       string
		paramName  string
		paramValue string
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "valid uuid",
			paramName:  "ID",
			paramValue: "123e4567-e89b-12d3-a456-426614174000",
			wantOK:     true,
			wantStatus: 0,
		},
		{
			name:       "invalid uuid format",
			paramName:  "ID",
			paramValue: "invalid-uuid",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty uuid",
			paramName:  "ID",
			paramValue: "",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)

			// Setup chi context
			rctx := chi.NewRouteContext()
			if tt.paramValue != "" {
				rctx.URLParams.Add(tt.paramName, tt.paramValue)
			}
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			result, ok := ParseuuidParam(w, r, tt.paramName)

			if ok != tt.wantOK {
				t.Errorf("ParseuuidParam() ok = %v, want %v", ok, tt.wantOK)
			}

			if !tt.wantOK {
				if w.Code != tt.wantStatus {
					t.Errorf("ParseuuidParam() status = %v, want %v", w.Code, tt.wantStatus)
				}
			} else {
				expecteduuid, err := uuid.Parse(tt.paramValue)
				if err != nil {
					t.Fatalf("uuid parse: %v", err)
				}
				if result != expecteduuid {
					t.Errorf("ParseuuidParam() = %v, want %v", result, expecteduuid)
				}
			}
		})
	}
}

//nolint:gocognit
func TestParseAuthUserID(t *testing.T) {
	validuuid := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		userID     string
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "valid user ID",
			userID:     validuuid,
			wantOK:     true,
			wantStatus: 0,
		},
		{
			name:       "invalid uuid format",
			userID:     "invalid-uuid",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty user ID",
			userID:     "",
			wantOK:     false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/test", nil)

			if tt.userID != "" {
				ctx := context.WithValue(r.Context(), UserIDKey, tt.userID)
				r = r.WithContext(ctx)
			}

			result, ok := ParseAuthUserID(w, r)

			if ok != tt.wantOK {
				t.Errorf("ParseAuthUserID() ok = %v, want %v", ok, tt.wantOK)
			}

			if !tt.wantOK {
				if w.Code != tt.wantStatus {
					t.Errorf("ParseAuthUserID() status = %v, want %v", w.Code, tt.wantStatus)
				}
			} else {
				expecteduuid, err := uuid.Parse(tt.userID)
				if err != nil {
					t.Fatalf("uuid parse: %v", err)
				}
				if result != expecteduuid {
					t.Errorf("ParseAuthUserID() = %v, want %v", result, expecteduuid)
				}
			}
		})
	}
}

func TestRenderJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	data := map[string]string{"message": "success"}
	RenderJSON(w, r, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("RenderJSON() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "success" {
		t.Errorf("RenderJSON() response = %v, want %v", response["message"], "success")
	}
}

func TestRenderNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	RenderNoContent(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("RenderNoContent() status = %v, want %v", w.Code, http.StatusNoContent)
	}
}

func TestRenderCreated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)

	data := map[string]string{"ID": "123"}
	RenderCreated(w, r, data)

	if w.Code != http.StatusCreated {
		t.Errorf("RenderCreated() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestRenderError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	RenderError(w, r, http.StatusNotFound, "not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("RenderError() status = %v, want %v", w.Code, http.StatusNotFound)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["error"] != "not found" {
		t.Errorf("RenderError() error = %v, want %v", response["error"], "not found")
	}
}

func TestRenderErrorWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	RenderErrorWithCode(w, r, http.StatusBadRequest, "invalid input", "INVALID_INPUT")

	if w.Code != http.StatusBadRequest {
		t.Errorf("RenderErrorWithCode() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["error"] != "invalid input" {
		t.Errorf("RenderErrorWithCode() error = %v, want %v", response["error"], "invalid input")
	}

	if response["code"] != "INVALID_INPUT" {
		t.Errorf("RenderErrorWithCode() code = %v, want %v", response["code"], "INVALID_INPUT")
	}
}
