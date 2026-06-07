package avatar

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	avatarHashLen = 16
)

const (
	imageMebibyte = 1024 * 1024

	formatMagicMinLen          = 4
	riffWebPHeaderLen          = 12
	webpMinAnimatedHeaderLen   = 16
	webpChunkHeaderLen         = 8
	webpChunkSizeOffset        = 4
	gifHeaderLen               = 6
	gifMinDataLen              = 13
	gifScreenDescriptorLen     = 7
	gifPackedFieldOffset       = 4
	gifImageDescriptorLen      = 9
	gifLocalPackedFieldOffset  = 8
	gifColorTableFlag          = 0x80
	gifColorTableSizeMask      = 0x07
	gifColorTableBase          = 2
	gifColorEntryBytes         = 3
	gifTrailerByte             = 0x3B
	gifImageDescriptorByte     = 0x2C
	gifExtensionIntroducerByte = 0x21
)

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

func (p *ImageProcessor) maxPixels() int {
	return p.cfg.MaxDimension * p.cfg.MaxDimension
}

// Process validates and transforms an uploaded image into a pair of WebP blobs.
// Security measures applied in order:
//  1. io.LimitReader caps the read at MaxFileSize+1.
//  2. Magic-byte validation rejects non-JPEG/PNG/GIF/WebP streams.
//  3. Animated GIF and animated WebP are rejected.
//  4. image.DecodeConfig checks dimension bounds without full decode.
//  5. Pixel-area guard prevents decompression-bomb attacks on extreme aspect ratios.
//
// The image is then center-cropped and scaled to FullSize and ThumbSize using
// Lanczos resampling and encoded as WebP at the configured quality. The
// SHA-256 hash of the full-size blob (truncated to 16 hex chars) is returned
// as the content-addressed storage key component.
func (p *ImageProcessor) Process(r io.Reader) (*domain.ProcessedAvatar, error) {
	data, err := io.ReadAll(io.LimitReader(r, p.cfg.MaxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("ImageProcessor - Process - read file: %w", err)
	}

	if int64(len(data)) > p.cfg.MaxFileSize {
		return nil, apperr.ErrFileTooLarge
	}

	format, ok := p.detectFormat(data)
	if !ok {
		return nil, apperr.ErrUnsupportedMediaType
	}

	if format == "gif" && isAnimatedGIF(data) {
		return nil, apperr.ErrAnimatedImageNotAllowed
	}

	if format == "webp" && isAnimatedWebP(data) {
		return nil, apperr.ErrAnimatedImageNotAllowed
	}

	cfg, err := decodeAvatarConfig(format, bytes.NewReader(data))
	if err != nil {
		return nil, apperr.NewValidationErrorf("invalid image file")
	}

	if cfg.Width < p.cfg.MinDimension || cfg.Height < p.cfg.MinDimension {
		return nil, apperr.NewValidationErrorf("image too small (min %dx%d)", p.cfg.MinDimension, p.cfg.MinDimension)
	}

	if cfg.Width > p.cfg.MaxDimension || cfg.Height > p.cfg.MaxDimension {
		return nil, apperr.NewValidationErrorf("image too large (max %dx%d)", p.cfg.MaxDimension, p.cfg.MaxDimension)
	}

	maxPixels := p.maxPixels()
	if cfg.Width*cfg.Height > maxPixels {
		return nil, apperr.NewValidationErrorf("image area too large (max %d Mpx)", maxPixels/imageMebibyte)
	}

	img, err := decodeAvatarImage(format, bytes.NewReader(data))
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

func decodeAvatarConfig(format string, r io.Reader) (image.Config, error) {
	switch format {
	case "jpeg":
		return jpeg.DecodeConfig(r)
	case "png":
		return png.DecodeConfig(r)
	case "gif":
		return gif.DecodeConfig(r)
	case "webp":
		return webp.DecodeConfig(r)
	default:
		return image.Config{}, fmt.Errorf("unsupported image format: %s", format)
	}
}

func decodeAvatarImage(format string, r io.Reader) (image.Image, error) {
	switch format {
	case "jpeg":
		return jpeg.Decode(r)
	case "png":
		return png.Decode(r)
	case "gif":
		return gif.Decode(r)
	case "webp":
		return webp.Decode(r)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}
}

// detectFormat checks the magic bytes of data and returns the format name
// ("jpeg", "png", "gif", "webp") and whether it is a recognised format.
func (p *ImageProcessor) detectFormat(data []byte) (string, bool) {
	if len(data) < formatMagicMinLen {
		return "", false
	}

	for format, magic := range allowedMagicBytes {
		if len(data) >= len(magic) && bytes.Equal(data[:len(magic)], magic) {
			return format, true
		}
	}

	if len(data) >= riffWebPHeaderLen &&
		string(data[:formatMagicMinLen]) == "RIFF" &&
		string(data[webpChunkHeaderLen:riffWebPHeaderLen]) == "WEBP" {
		return "webp", true
	}

	return "", false
}

// isAnimatedGIF returns true if data contains a GIF with more than one Image
// Descriptor block (i.e., multiple frames). It parses the GIF block structure
// without decoding pixel data to avoid decompression-bomb amplification.
func isAnimatedGIF(data []byte) bool {
	if len(data) < gifMinDataLen {
		return false
	}

	pos := gifHeaderLen // skip "GIF87a" / "GIF89a" signature

	if pos+gifScreenDescriptorLen > len(data) {
		return false
	}

	globalFlags := data[pos+gifPackedFieldOffset]
	pos += gifScreenDescriptorLen

	// Skip Global Color Table
	if globalFlags&gifColorTableFlag != 0 {
		colorCount := gifColorTableBase << (globalFlags & gifColorTableSizeMask)
		pos += colorCount * gifColorEntryBytes
	}

	frameCount := 0

	for pos < len(data) {
		switch data[pos] {
		case gifTrailerByte:
			return frameCount > 1

		case gifImageDescriptorByte:
			frameCount++
			if frameCount > 1 {
				return true
			}

			pos++
			if pos+gifImageDescriptorLen > len(data) {
				return false
			}

			localFlags := data[pos+gifLocalPackedFieldOffset]
			pos += gifImageDescriptorLen

			// Skip Local Color Table
			if localFlags&gifColorTableFlag != 0 {
				colorCount := gifColorTableBase << (localFlags & gifColorTableSizeMask)
				pos += colorCount * gifColorEntryBytes
			}

			// Skip LZW Minimum Code Size byte
			pos++
			// Skip sub-blocks
			pos = skipGIFSubBlocks(data, pos)

		case gifExtensionIntroducerByte:
			pos += 2 // skip introducer + label
			pos = skipGIFSubBlocks(data, pos)

		default:
			// Unknown block - stop parsing
			return false
		}
	}

	return false
}

// skipGIFSubBlocks advances pos past a series of GIF sub-blocks (each prefixed
// by a length byte, terminated by a zero-length block).
func skipGIFSubBlocks(data []byte, pos int) int {
	for pos < len(data) {
		size := int(data[pos])
		pos++

		if size == 0 {
			break
		}

		pos += size
	}

	return pos
}

// isAnimatedWebP returns true if the WebP file contains an ANIM chunk, which
// is the definitive indicator of WebP animation per the WebP spec.
func isAnimatedWebP(data []byte) bool {
	// RIFF header: "RIFF"(4) + fileSize(4) + "WEBP"(4) = 12 bytes minimum
	if len(data) < webpMinAnimatedHeaderLen {
		return false
	}

	pos := riffWebPHeaderLen // start after RIFF/fileSize/WEBP header

	for pos+webpChunkHeaderLen <= len(data) {
		chunkID := string(data[pos : pos+webpChunkSizeOffset])
		if chunkID == "ANIM" {
			return true
		}

		chunkSize := int(binary.LittleEndian.Uint32(data[pos+webpChunkSizeOffset : pos+webpChunkHeaderLen]))
		pos += webpChunkHeaderLen + chunkSize

		// WebP chunks are padded to even byte boundary
		if chunkSize%2 != 0 {
			pos++
		}
	}

	return false
}
