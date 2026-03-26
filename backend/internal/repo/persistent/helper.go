package persistent

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func EnsureID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

func GetOrNotFound[T any](fn func() (T, error), notFoundErr *httperr.HTTPError, op string) (T, error) {
	var zero T

	v, err := fn()
	if err != nil {
		if pgutil.IsNoRows(err) {
			return zero, notFoundErr
		}

		return zero, fmt.Errorf("%s: %w", op, err)
	}

	return v, nil
}

func timeFromNullableAny(v any) time.Time {
	if v == nil {
		return time.Time{}
	}

	if t, ok := v.(*time.Time); ok && t != nil {
		return *t
	}

	if t, ok := v.(time.Time); ok {
		return t
	}

	return time.Time{}
}

func intToInt32Safe(i int) (int32, error) {
	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0, fmt.Errorf("int value %d out of int32 range", i)
	}

	return int32(i), nil
}

type IntField struct {
	Name  string
	Value int
}

func ConvertIntFieldsToInt32(fields []IntField) ([]int32, error) {
	out := make([]int32, 0, len(fields))
	for _, f := range fields {
		v, err := intToInt32Safe(f.Value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}

		out = append(out, v)
	}

	return out, nil
}

func toLimitOffset(limit, offset int) (int32, int32, error) {
	l, err := intToInt32Safe(limit)
	if err != nil {
		return 0, 0, fmt.Errorf("toLimitOffset limit: %w", err)
	}

	o, err := intToInt32Safe(offset)
	if err != nil {
		return 0, 0, fmt.Errorf("toLimitOffset offset: %w", err)
	}

	return l, o, nil
}

func intToInt32Ptr(i int) (*int32, error) {
	if i == 0 {
		return nil, nil
	}

	v, err := intToInt32Safe(i)
	if err != nil {
		return nil, fmt.Errorf("intToInt32Ptr: %w", err)
	}

	return &v, nil
}

func EscapeLikePattern(s string) string {
	replacer := strings.NewReplacer(
		"%", `\%`,
		"_", `\_`,
		`\`, `\\`,
	)

	return replacer.Replace(s)
}
