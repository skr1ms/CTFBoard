package entityError

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPError_Error_Success(t *testing.T) {
	err := errors.New("test message")
	he := &HTTPError{Err: err, StatusCode: 404, Code: "not_found"}
	assert.Equal(t, "test message", he.Error())
}

func TestHTTPError_Unwrap_Success(t *testing.T) {
	err := errors.New("wrapped")
	he := &HTTPError{Err: err, StatusCode: 500, Code: "internal"}
	assert.Same(t, err, he.Unwrap())
}

func TestHTTPError_HTTPStatus_Success(t *testing.T) {
	he := &HTTPError{Err: errors.New("x"), StatusCode: 403, Code: "forbidden"}
	assert.Equal(t, 403, he.HTTPStatus())
}

func TestHTTPError_HTTPStatus_DifferentCode(t *testing.T) {
	he := &HTTPError{Err: errors.New("x"), StatusCode: 401, Code: "unauthorized"}
	assert.Equal(t, 401, he.HTTPStatus())
}
