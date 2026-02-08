package usecaseutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrap_Success(t *testing.T) {
	got := Wrap(nil, "msg")
	assert.NoError(t, got)
	assert.Nil(t, got)
}

func TestWrap_Error(t *testing.T) {
	err := errors.New("original")
	got := Wrap(err, "context")
	assert.Error(t, got)
	assert.Contains(t, got.Error(), "context")
	assert.Contains(t, got.Error(), "original")
	assert.True(t, errors.Is(got, err))
}
