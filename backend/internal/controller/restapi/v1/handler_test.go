package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/helper"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLogger struct{}

func (stubLogger) Debug(msg string, fields ...logger.Fields)     {}
func (stubLogger) Info(msg string, fields ...logger.Fields)      {}
func (stubLogger) Warn(msg string, fields ...logger.Fields)      {}
func (stubLogger) Error(msg string, fields ...logger.Fields)     {}
func (stubLogger) Fatal(msg string, fields ...logger.Fields)     {}
func (stubLogger) WithFields(fields logger.Fields) logger.Logger { return stubLogger{} }
func (stubLogger) WithError(err error) logger.Logger             { return stubLogger{} }

func testServer() *Server {
	return &Server{
		infra: helper.InfraDeps{Logger: stubLogger{}},
	}
}

func TestOK(t *testing.T) {
	res := OK(map[string]string{"k": "v"})
	assert.Equal(t, http.StatusOK, res.Status)
	assert.Equal(t, map[string]string{"k": "v"}, res.Data)
}

func TestCreated(t *testing.T) {
	res := Created(42)
	assert.Equal(t, http.StatusCreated, res.Status)
	assert.Equal(t, 42, res.Data)
}

func TestNoContent(t *testing.T) {
	res := NoContent()
	assert.Equal(t, http.StatusNoContent, res.Status)
	assert.Nil(t, res.Data)
}

func TestHandle_Error(t *testing.T) {
	h := testServer()
	handled := h.Handle("TestOp", func(w http.ResponseWriter, r *http.Request) (HandlerResult, error) {
		return HandlerResult{}, errors.New("test err")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handled(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "Internal server error", body["error"])
}

func TestHandle_NoContent(t *testing.T) {
	h := testServer()
	handled := h.Handle("NoContent", func(w http.ResponseWriter, r *http.Request) (HandlerResult, error) {
		return NoContent(), nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handled(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 0, w.Body.Len())
}

func TestHandle_Created(t *testing.T) {
	h := testServer()
	handled := h.Handle("Created", func(w http.ResponseWriter, r *http.Request) (HandlerResult, error) {
		return Created(map[string]int{"id": 1}), nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	handled(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body map[string]int
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, 1, body["id"])
}

func TestHandle_OK(t *testing.T) {
	h := testServer()
	handled := h.Handle("OK", func(w http.ResponseWriter, r *http.Request) (HandlerResult, error) {
		return OK(map[string]bool{"ok": true}), nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handled(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.True(t, body["ok"])
}
