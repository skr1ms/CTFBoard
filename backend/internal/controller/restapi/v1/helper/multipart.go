package helper

import (
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func ParseMultipartFormLimit(w http.ResponseWriter, r *http.Request, maxMemory int64) bool {
	return httputil.ParseMultipartFormLimit(w, r, maxMemory)
}

func ReadFormFile(w http.ResponseWriter, r *http.Request, key string, maxSize int64) ([]byte, *multipart.FileHeader, bool) {
	return httputil.ReadFormFile(w, r, key, maxSize)
}

// DecodeMultipartForm populates a struct from a parsed multipart form. Fields are matched
// by their JSON struct tag name. Supported field types: *bool, *string, string,
// openapi_types.File (populated via multipart.FileHeader).
func DecodeMultipartForm[T any](r *http.Request, dst *T) {
	if r.MultipartForm == nil {
		return
	}

	v := reflect.ValueOf(dst).Elem()
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		fv := v.Field(i)

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]

		switch fv.Type() {
		case reflect.TypeOf((*bool)(nil)):
			if val := r.FormValue(name); val != "" {
				b := val == "true"
				fv.Set(reflect.ValueOf(&b))
			}
		case reflect.TypeOf(""):
			if val := r.FormValue(name); val != "" {
				fv.SetString(val)
			}
		case reflect.TypeOf((*string)(nil)):
			if val := r.FormValue(name); val != "" {
				s := val
				fv.Set(reflect.ValueOf(&s))
			}
		case reflect.TypeOf(openapi_types.File{}):
			if headers, ok := r.MultipartForm.File[name]; ok && len(headers) > 0 {
				var f openapi_types.File
				f.InitFromMultipart(headers[0])
				fv.Set(reflect.ValueOf(f))
			}
		default:
			// For named string types (enums): underlying kind is string.
			if fv.Kind() == reflect.String {
				if val := r.FormValue(name); val != "" {
					fv.SetString(val)
				}
			} else if fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.String {
				if val := r.FormValue(name); val != "" {
					ptr := reflect.New(fv.Type().Elem())
					ptr.Elem().SetString(val)
					fv.Set(ptr)
				}
			}
		}
	}
}
