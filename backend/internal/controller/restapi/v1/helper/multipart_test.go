package helper

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type multipartTestStruct struct {
	Name    string             `json:"name"`
	NamePtr *string            `json:"name_ptr"`
	Flag    bool               `json:"-"`
	Active  *bool              `json:"active"`
	Ignored string             `json:"-"`
	File    openapi_types.File `json:"file"`
}

type namedStringType string

type multipartEnumStruct struct {
	Mode    namedStringType  `json:"mode"`
	ModePtr *namedStringType `json:"mode_ptr"`
}

func newMultipartRequest(t *testing.T, build func(w *multipart.Writer)) *http.Request {
	t.Helper()

	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)
	build(w)
	require.NoError(t, w.Close())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(32<<20))

	return req
}

func TestDecodeMultipartForm_StringField(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("name", "hello"))
	})

	var dst multipartTestStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	assert.Equal(t, "hello", dst.Name)
}

func TestDecodeMultipartForm_StringPtrField(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("name_ptr", "world"))
	})

	var dst multipartTestStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	require.NotNil(t, dst.NamePtr)
	assert.Equal(t, "world", *dst.NamePtr)
}

func TestDecodeMultipartForm_BoolPtrField_True(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("active", "true"))
	})

	var dst multipartTestStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	require.NotNil(t, dst.Active)
	assert.True(t, *dst.Active)
}

func TestDecodeMultipartForm_BoolPtrField_Invalid(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("active", "invalid"))
	})

	var dst multipartTestStruct

	err := DecodeMultipartForm(req, &dst, nil)
	require.Error(t, err)
}

func TestDecodeMultipartForm_BoolPtrField_False(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("active", "false"))
	})

	var dst multipartTestStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	require.NotNil(t, dst.Active)
	assert.False(t, *dst.Active)
}

func TestDecodeMultipartForm_SkipsJSONDashTag(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("-", "should-be-ignored"))
	})

	var dst multipartTestStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	assert.Empty(t, dst.Ignored)
}

func TestDecodeMultipartForm_NoMultipartForm_IsNoOp(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", http.NoBody)
	require.NoError(t, err)

	var dst multipartTestStruct

	dst.Name = "original"
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	assert.Equal(t, "original", dst.Name)
}

func TestDecodeMultipartForm_FileField(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="file"; filename="test.txt"`)
		h.Set("Content-Type", "text/plain")
		fw, err := w.CreatePart(h)
		require.NoError(t, err)
		_, err = fw.Write([]byte("file contents"))
		require.NoError(t, err)
	})

	var dst multipartTestStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	assert.NotEmpty(t, dst.File.Filename())
}

func TestDecodeMultipartForm_NamedStringType(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("mode", "ctf"))
	})

	var dst multipartEnumStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	assert.Equal(t, namedStringType("ctf"), dst.Mode)
}

func TestDecodeMultipartForm_NamedStringPtrType(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("mode_ptr", "jeopardy"))
	})

	var dst multipartEnumStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	require.NotNil(t, dst.ModePtr)
	assert.Equal(t, namedStringType("jeopardy"), *dst.ModePtr)
}

func TestDecodeMultipartForm_MultipleFields(t *testing.T) {
	t.Parallel()
	req := newMultipartRequest(t, func(w *multipart.Writer) {
		require.NoError(t, w.WriteField("name", "alice"))
		require.NoError(t, w.WriteField("active", "true"))
		require.NoError(t, w.WriteField("name_ptr", "bob"))
	})

	var dst multipartTestStruct
	require.NoError(t, DecodeMultipartForm(req, &dst, nil))
	assert.Equal(t, "alice", dst.Name)
	require.NotNil(t, dst.Active)
	assert.True(t, *dst.Active)
	require.NotNil(t, dst.NamePtr)
	assert.Equal(t, "bob", *dst.NamePtr)
}
