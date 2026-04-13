package persistent

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"
)

// EnsureID generates a new UUID and assigns it to *id if it is currently uuid.Nil.
// Called in Create methods to allow callers to pre-set an ID or rely on auto-generation.
func EnsureID(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

// GetOrNotFound executes fn and maps pgx ErrNoRows to the provided domain
// sentinel error, wrapping all other errors with the op prefix for context.
func GetOrNotFound[T any](fn func() (T, error), notFoundErr error, op string) (T, error) {
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

// timeFromNullableAny coerces a dynamically typed value returned by the
// pgx generic-query path into a time.Time. Handles nil, *time.Time, and
// time.Time; returns the zero value for all other types.
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
		return 0, fmt.Errorf("intToInt32Safe - out of int32 range: int value %d", i)
	}

	return int32(i), nil
}

type IntField struct {
	Name  string
	Value int
}

// ConvertIntFieldsToInt32 narrows a slice of named int fields to int32, returning an error
// that includes the field name if any value is outside the int32 range.
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
		return 0, 0, fmt.Errorf("toLimitOffset - limit: %w", err)
	}

	o, err := intToInt32Safe(offset)
	if err != nil {
		return 0, 0, fmt.Errorf("toLimitOffset - offset: %w", err)
	}

	return l, o, nil
}

func intToInt32Ptr(i int) (*int32, error) {
	if i == 0 {
		return nil, nil
	}

	v, err := intToInt32Safe(i)
	if err != nil {
		return nil, fmt.Errorf("intToInt32Ptr - convert: %w", err)
	}

	return &v, nil
}

// EscapeLikePattern escapes the three PostgreSQL LIKE metacharacters (%_\) so
// that caller-supplied search strings are treated as literals in LIKE queries.
func EscapeLikePattern(s string) string {
	replacer := strings.NewReplacer(
		"%", `\%`,
		"_", `\_`,
		`\`, `\\`,
	)

	return replacer.Replace(s)
}

// EscapeSearchPtr escapes a nullable search term for use in a LIKE query.
// Returns nil when search is nil, otherwise returns a pointer to the escaped string.
func EscapeSearchPtr(search *string) *string {
	if search == nil {
		return nil
	}

	escaped := EscapeLikePattern(*search)

	return &escaped
}
