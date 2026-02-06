package httputil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeJSON_Success(t *testing.T) {
	body := map[string]string{"key": "value"}
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	r := httptest.NewRequest("POST", "/", bytes.NewReader(jsonBody))
	var result map[string]string
	err = DecodeJSON(r, &result)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

func TestDecodeJSON_Error(t *testing.T) {
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte("invalid json")))
	var result map[string]string
	err := DecodeJSON(r, &result)
	assert.Error(t, err)
}

func TestDecodeAndValidate_Success(t *testing.T) {
	type Req struct {
		Name string `validate:"not_empty"`
	}
	body := Req{Name: "test"}
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader(jsonBody))
	v := validator.New()
	l := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})
	got, ok := DecodeAndValidate[Req](w, r, v, l, "test")
	require.True(t, ok)
	assert.Equal(t, "test", got.Name)
}

func TestDecodeAndValidate_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte("{")))
	v := validator.New()
	l := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})
	_, ok := DecodeAndValidate[struct {
		Name string `validate:"not_empty"`
	}](w, r, v, l, "test")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDecodeAndValidate_ValidationError(t *testing.T) {
	type Req struct {
		Name string `validate:"not_empty"`
	}
	body := Req{Name: ""}
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", bytes.NewReader(jsonBody))
	v := validator.New()
	l := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})
	_, ok := DecodeAndValidate[Req](w, r, v, l, "test")
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
