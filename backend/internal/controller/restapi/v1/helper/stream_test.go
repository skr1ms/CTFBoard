package helper

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareChallengeUploadReaderAllowsCTFBinaryContent(t *testing.T) {
	t.Parallel()

	content := []byte("\x7fELF\x02\x01ctf binary")

	reader, contentType, err := PrepareChallengeUploadReader(bytes.NewReader(content), "chall.elf")
	require.NoError(t, err)

	got, err := io.ReadAll(reader)
	require.NoError(t, err)

	assert.Equal(t, content, got)
	assert.Equal(t, "application/octet-stream", contentType)
}

func TestPrepareUploadReaderStillBlocksDangerousMagicBytes(t *testing.T) {
	t.Parallel()

	reader, contentType, err := PrepareUploadReader(bytes.NewReader([]byte("\x7fELF\x02\x01")), "avatar.webp")

	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Empty(t, contentType)
}
