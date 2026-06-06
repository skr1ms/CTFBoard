package helper

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/storagepath"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const (
	binaryContentType       = "application/octet-stream"
	contentTypePeekSize     = 512
	uploadMagicBytePeekSize = 8
	noSniffHeaderName       = "X-Content-Type-Options"
	noSniffHeaderValue      = "nosniff"
)

func DetectContentType(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return binaryContentType
	}

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return binaryContentType
	}

	return contentType
}

// DetectContentTypeFromReader peeks at a stream for MIME sniffing and returns a
// replacement reader that preserves the bytes consumed during the peek.
func DetectContentTypeFromReader(filename string, r io.Reader) (string, io.Reader) {
	peek := make([]byte, contentTypePeekSize)
	n, err := r.Read(peek)
	peek = peek[:n]
	body := io.MultiReader(bytes.NewReader(peek), r)

	if err != nil && !errors.Is(err, io.EOF) {
		return DetectContentType(filename), body
	}

	if n > 0 {
		detected := http.DetectContentType(peek)
		if detected != "" && detected != binaryContentType {
			return detected, body
		}
	}

	return DetectContentType(filename), body
}

// PrepareUploadReader validates high-risk magic bytes while preserving the
// original stream for the usecase upload path.
func PrepareUploadReader(r io.Reader, filename string) (io.Reader, string, error) {
	peek := make([]byte, uploadMagicBytePeekSize)

	n, err := io.ReadFull(r, peek)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, "", err
	}

	if n > 0 && !validator.ValidateFileMagic(peek[:n]) {
		return nil, "", apperr.NewValidationErrorf("file type not allowed")
	}

	return io.MultiReader(bytes.NewReader(peek[:n]), r), DetectContentType(filename), nil
}

func ValidateDownloadPath(path string) bool {
	return storagepath.ValidateDownloadPath(path)
}

func DownloadFilename(path string) string {
	return storagepath.DownloadFilename(path)
}

func DownloadPathFromWildcard(r *http.Request) string {
	return chi.URLParam(r, "*")
}

func DownloadToken(r *http.Request) string {
	return r.URL.Query().Get("token")
}

func RenderDownloadStream(w http.ResponseWriter, contentType, filename string, body io.Reader) error {
	w.Header().Set(noSniffHeaderName, noSniffHeaderValue)

	return httputil.RenderStream(w, contentType, filename, body)
}

func RenderCachedImage(w http.ResponseWriter, r *http.Request, reader io.Reader, contentType, cacheControl, etag string) error {
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)

		return nil
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("ETag", etag)
	w.Header().Set(noSniffHeaderName, noSniffHeaderValue)
	w.Header().Set("Content-Security-Policy", "default-src 'none'")

	_, err := io.Copy(w, reader)

	return err
}
