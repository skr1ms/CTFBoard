package avatar

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const avatarHashLen = 16

var allowedMagicBytes = map[string][]byte{
	"jpeg": {0xFF, 0xD8, 0xFF},
	"png":  {0x89, 0x50, 0x4E, 0x47},
	"gif":  {0x47, 0x49, 0x46, 0x38},
}

type ImageProcessor struct {
	cfg domain.AvatarConfig
}

func NewImageProcessor(cfg domain.AvatarConfig) *ImageProcessor {
	return &ImageProcessor{cfg: cfg}
}

// Process validates and transforms an uploaded image into a pair of WebP blobs.
// Security measures applied in order: io.LimitReader caps the read at MaxFileSize+1
// to prevent memory exhaustion; magic-byte validation rejects non-JPEG/PNG/GIF/WebP
// streams; image.DecodeConfig checks dimension bounds without full decode. The image
// is then center-cropped and scaled to FullSize and ThumbSize using Lanczos
// resampling and encoded as WebP at the configured quality. The SHA-256 hash of the
// full-size blob (truncated to 16 hex chars) is returned as the content-addressed
// storage key component.
func (p *ImageProcessor) Process(r io.Reader) (*domain.ProcessedAvatar, error) {
	data, err := io.ReadAll(io.LimitReader(r, p.cfg.MaxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("ImageProcessor - Process - read file: %w", err)
	}

	if int64(len(data)) > p.cfg.MaxFileSize {
		return nil, apperr.ErrFileTooLarge
	}

	if !p.validateMagicBytes(data) {
		return nil, apperr.ErrUnsupportedMediaType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, apperr.NewValidationErrorf("invalid image file")
	}

	if cfg.Width < p.cfg.MinDimension || cfg.Height < p.cfg.MinDimension {
		return nil, apperr.NewValidationErrorf("image too small (min %dx%d)", p.cfg.MinDimension, p.cfg.MinDimension)
	}

	if cfg.Width > p.cfg.MaxDimension || cfg.Height > p.cfg.MaxDimension {
		return nil, apperr.NewValidationErrorf("image too large (max %dx%d)", p.cfg.MaxDimension, p.cfg.MaxDimension)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, apperr.NewValidationErrorf("failed to decode image")
	}

	full := imaging.Fill(img, p.cfg.FullSize, p.cfg.FullSize, imaging.Center, imaging.Lanczos)
	thumb := imaging.Fill(img, p.cfg.ThumbSize, p.cfg.ThumbSize, imaging.Center, imaging.Lanczos)

	var fullBuf, thumbBuf bytes.Buffer
	if err := webp.Encode(&fullBuf, full, &webp.Options{Quality: float32(p.cfg.WebPQuality)}); err != nil {
		return nil, fmt.Errorf("ImageProcessor - Process - encode full webp: %w", err)
	}

	if err := webp.Encode(&thumbBuf, thumb, &webp.Options{Quality: float32(p.cfg.WebPQuality)}); err != nil {
		return nil, fmt.Errorf("ImageProcessor - Process - encode thumb webp: %w", err)
	}

	hash := sha256.Sum256(fullBuf.Bytes())
	hashStr := hex.EncodeToString(hash[:])

	return &domain.ProcessedAvatar{
		FullImage:      fullBuf.Bytes(),
		ThumbnailImage: thumbBuf.Bytes(),
		SHA256Hash:     hashStr[:avatarHashLen],
	}, nil
}

func (p *ImageProcessor) validateMagicBytes(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	for _, magic := range allowedMagicBytes {
		if len(data) >= len(magic) && bytes.Equal(data[:len(magic)], magic) {
			return true
		}
	}

	if len(data) >= 12 &&
		string(data[:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP" {
		return true
	}

	return false
}
