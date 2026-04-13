package helper

import (
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/form/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

var multipartFormDecoder = func() *form.Decoder {
	d := form.NewDecoder()
	d.SetTagName("json")

	return d
}()

type multipartErrorReporter func(w http.ResponseWriter, r *http.Request, err error, op, step string) bool

func RequireMultipartFile(w http.ResponseWriter, r *http.Request, onError multipartErrorReporter, op, step string, fileSize int64) bool {
	if fileSize == 0 {
		onError(w, r, apperr.NewValidationErrorf("file is required"), op, step)

		return false
	}

	return true
}

// OpenAvatarFile validates that the file is non-empty and opens it for reading.
// Returns the ReadCloser and true on success; writes an error response and
// returns false on failure. The caller is responsible for closing the reader.
func OpenAvatarFile(w http.ResponseWriter, r *http.Request, onError multipartErrorReporter, op string, f *openapi_types.File) (io.ReadCloser, bool) {
	if !RequireMultipartFile(w, r, onError, op, "FormFile", f.FileSize()) {
		return nil, false
	}

	rc, err := f.Reader()
	if onError(w, r, err, op, "OpenFile") {
		return nil, false
	}

	return rc, true
}

func ValidateMultipartEnum(fieldName, value string, allowed []string) error {
	if lo.Contains(allowed, value) {
		return nil
	}

	return apperr.NewValidationErrorf("invalid %s: allowed values are %s", fieldName, strings.Join(allowed, ", "))
}

func ParseMultipartFormLimit(w http.ResponseWriter, r *http.Request, maxBodySize, maxMemory int64) bool {
	return httputil.ParseMultipartFormLimit(w, r, maxBodySize, maxMemory)
}

const maxMultipartStringFieldLen = 10_000

// checkMultipartValueLengths rejects multipart string fields exceeding
// maxMultipartStringFieldLen bytes. This is a DoS guard - large string fields
// would otherwise be decoded into memory before the validator can inspect them.
func checkMultipartValueLengths(vals map[string][]string) error {
	for name, list := range vals {
		for _, v := range list {
			if len(v) > maxMultipartStringFieldLen {
				return apperr.NewValidationErrorf("%s exceeds maximum length", name)
			}
		}
	}

	return nil
}

// setMultipartFileFields populates openapi_types.File fields in dst by matching
// each field's JSON struct tag name against r.MultipartForm.File. Uses reflection
// so that new file fields added to the multipart body struct are handled automatically
// without modifying this function.
func setMultipartFileFields(r *http.Request, dst reflect.Value) {
	t := dst.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type != reflect.TypeFor[openapi_types.File]() {
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		name := strings.Split(tag, ",")[0]

		headers, ok := r.MultipartForm.File[name]
		if !ok || len(headers) == 0 {
			continue
		}

		var file openapi_types.File
		file.InitFromMultipart(headers[0])
		dst.Field(i).Set(reflect.ValueOf(file))
	}
}

// DecodeMultipartForm populates a struct from a parsed multipart form. Fields are matched
// by their JSON struct tag name. Values are read only from MultipartForm.Value (not query)
// If v is non-nil, validator.Validate(dst) is called after decoding and its error is returned.
func DecodeMultipartForm[T any](r *http.Request, dst *T, v validator.Validator) error {
	if r.MultipartForm == nil {
		return nil
	}

	vals := r.MultipartForm.Value

	err := checkMultipartValueLengths(vals)
	if err != nil {
		return err
	}

	err = multipartFormDecoder.Decode(dst, vals)
	if err != nil {
		return apperr.NewValidationErrorf("%v", err)
	}

	rv := reflect.ValueOf(dst).Elem()
	setMultipartFileFields(r, rv)

	if v != nil {
		return v.Validate(dst)
	}

	return nil
}
