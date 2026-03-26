package avatar

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestImageProcessor_Process_ValidPNG(t *testing.T) {
	cfg := domain.GetDefaultAvatarConfig()
	processor := NewImageProcessor(cfg)

	img := image.NewRGBA(image.Rect(0, 0, 512, 512))

	for y := range 512 {
		for x := range 512 {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	result, err := processor.Process(&buf)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.FullImage)
	assert.NotEmpty(t, result.ThumbnailImage)
	assert.Len(t, result.SHA256Hash, 16)
}

func TestImageProcessor_Process_InvalidMagicBytes(t *testing.T) {
	cfg := domain.GetDefaultAvatarConfig()
	processor := NewImageProcessor(cfg)

	invalidData := []byte{0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF}
	buf := bytes.NewReader(invalidData)

	result, err := processor.Process(buf)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestImageProcessor_Process_TooSmall(t *testing.T) {
	cfg := domain.GetDefaultAvatarConfig()
	processor := NewImageProcessor(cfg)

	img := image.NewRGBA(image.Rect(0, 0, 32, 32))

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	result, err := processor.Process(&buf)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "too small")
}

func TestImageProcessor_Process_TooLarge(t *testing.T) {
	cfg := domain.GetDefaultAvatarConfig()
	cfg.MaxDimension = 1024
	processor := NewImageProcessor(cfg)

	img := image.NewRGBA(image.Rect(0, 0, 5000, 5000))

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	result, err := processor.Process(&buf)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "too large")
}
