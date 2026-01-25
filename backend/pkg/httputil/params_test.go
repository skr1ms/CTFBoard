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

func TestParseUUIDParam(t *testing.T) {
	tests := []struct {
		name       string
		paramName  string
		paramValue string
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "valid UUID",
			paramName:  "id",
			paramValue: "123e4567-e89b-12d3-a456-426614174000",
			wantOK:     true,
			wantStatus: 0,
		},
		{
			name:       "invalid UUID format",
			paramName:  "id",
			paramValue: "invalid-uuid",
			wantOK:     false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty UUID",
			paramName:  "id",
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

			result, ok := ParseUUIDParam(w, r, tt.paramName)

			if ok != tt.wantOK {
				t.Errorf("ParseUUIDParam() ok = %v, want %v", ok, tt.wantOK)
			}

			if !tt.wantOK {
				if w.Code != tt.wantStatus {
					t.Errorf("ParseUUIDParam() status = %v, want %v", w.Code, tt.wantStatus)
				}
			} else {
				expectedUUID, _ := uuid.Parse(tt.paramValue)
				if result != expectedUUID {
					t.Errorf("ParseUUIDParam() = %v, want %v", result, expectedUUID)
				}
			}
		})
	}
}

func TestParseAuthUserID(t *testing.T) {
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	tests := []struct {
		name       string
		userID     string
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "valid user ID",
			userID:     validUUID,
			wantOK:     true,
			wantStatus: 0,
		},
		{
			name:       "invalid UUID format",
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
				expectedUUID, _ := uuid.Parse(tt.userID)
				if result != expectedUUID {
					t.Errorf("ParseAuthUserID() = %v, want %v", result, expectedUUID)
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

	data := map[string]string{"id": "123"}
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
