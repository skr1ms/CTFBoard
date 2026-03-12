package helper

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type multipartErrorReporter func(w http.ResponseWriter, r *http.Request, err error, op, step string) bool

func RequireMultipartFile(w http.ResponseWriter, r *http.Request, onError multipartErrorReporter, op, step string, fileSize int64) bool {
	if fileSize == 0 {
		onError(w, r, NewValidationErrorf("file is required"), op, step)
		return false
	}
	return true
}

func ValidateMultipartEnum(fieldName, value string, allowed []string) error {
	for _, a := range allowed {
		if a == value {
			return nil
		}
	}
	return NewValidationErrorf("invalid %s: allowed values are %s", fieldName, strings.Join(allowed, ", "))
}

func ParseMultipartFormLimit(w http.ResponseWriter, r *http.Request, maxMemory int64) bool {
	return httputil.ParseMultipartFormLimit(w, r, maxMemory)
}

const maxMultipartStringFieldLen = 10_000

func multipartFormValue(form map[string][]string, name string) string {
	if form == nil {
		return ""
	}
	vals := form[name]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// DecodeMultipartForm populates a struct from a parsed multipart form. Fields are matched
// by their JSON struct tag name. Values are read only from MultipartForm.Value (not query).
// If v is non-nil, validator.Validate(dst) is called after decoding and its error is returned.
func DecodeMultipartForm[T any](r *http.Request, dst *T, v validator.Validator) error {
	if r.MultipartForm == nil {
		return nil
	}
	vals := r.MultipartForm.Value
	rv := reflect.ValueOf(dst).Elem()
	t := rv.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		fv := rv.Field(i)

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		val := multipartFormValue(vals, name)

		switch fv.Type() {
		case reflect.TypeOf((*bool)(nil)):
			if val != "" {
				b, err := strconv.ParseBool(val)
				if err != nil {
					return NewValidationErrorf("%s must be true or false", name)
				}
				fv.Set(reflect.ValueOf(&b))
			}
		case reflect.TypeOf(""):
			if val != "" {
				if len(val) > maxMultipartStringFieldLen {
					return NewValidationErrorf("%s exceeds maximum length", name)
				}
				fv.SetString(val)
			}
		case reflect.TypeOf((*string)(nil)):
			if val != "" {
				if len(val) > maxMultipartStringFieldLen {
					return NewValidationErrorf("%s exceeds maximum length", name)
				}
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
			if fv.Kind() == reflect.String {
				if val != "" {
					if len(val) > maxMultipartStringFieldLen {
						return NewValidationErrorf("%s exceeds maximum length", name)
					}
					fv.SetString(val)
				}
			} else if fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.String {
				if val != "" {
					if len(val) > maxMultipartStringFieldLen {
						return NewValidationErrorf("%s exceeds maximum length", name)
					}
					ptr := reflect.New(fv.Type().Elem())
					ptr.Elem().SetString(val)
					fv.Set(ptr)
				}
			}
		}
	}
	if v != nil {
		return v.Validate(dst)
	}
	return nil
}
