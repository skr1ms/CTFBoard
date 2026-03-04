package helper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
)

func RenderOK[T any](w http.ResponseWriter, r *http.Request, data T) {
	httputil.RenderOK(w, r, data)
}

func RenderCreated[T any](w http.ResponseWriter, r *http.Request, data T) {
	httputil.RenderCreated(w, r, data)
}

func RenderNoContent(w http.ResponseWriter, r *http.Request) {
	httputil.RenderNoContent(w, r)
}

func RenderJSON[T any](w http.ResponseWriter, r *http.Request, status int, data T) {
	httputil.RenderJSON(w, r, status, data)
}

func RenderText(w http.ResponseWriter, r *http.Request, status int, contentType, body string) {
	httputil.RenderText(w, r, status, contentType, body)
}

// RenderJSONAttachment encodes data as JSON and writes it as a downloadable
// attachment. It encodes into a buffer first so headers are only sent after a
// successful encode, avoiding partial responses on error. Returns an error if
// encoding or writing fails.
func RenderJSONAttachment[T any](w http.ResponseWriter, r *http.Request, data T, filename string) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return fmt.Errorf("encode json attachment: %w", err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	_, err := w.Write(buf.Bytes())
	return err
}

// RenderStream writes a streaming response with the given content type and
// attachment filename. The caller is responsible for closing rc.
func RenderStream(w http.ResponseWriter, contentType, filename string, rc io.Reader) error {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	_, err := io.Copy(w, rc)
	return err
}

// RenderBytes writes a raw byte slice with the given content type and
// attachment filename.
func RenderBytes(w http.ResponseWriter, contentType, filename string, data []byte) error {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	_, err := w.Write(data)
	return err
}
