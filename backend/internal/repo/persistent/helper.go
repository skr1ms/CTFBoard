package persistent

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func isNoRows(err error) bool {
	return err != nil && errors.Is(err, pgx.ErrNoRows)
}

func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return err != nil && errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func ptrTimeToTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func timeFromNullable(v any) time.Time {
	if v == nil {
		return time.Time{}
	}
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Time{}
}

func int32PtrToInt(p *int32) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func boolPtrToBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func intToInt32Safe(i int) (int32, error) {
	if i < math.MinInt32 || i > math.MaxInt32 {
		return 0, fmt.Errorf("int value %d out of int32 range", i)
	}
	return int32(i), nil
}

func intToInt32Ptr(i int) *int32 {
	if i == 0 {
		return nil
	}
	v, err := intToInt32Safe(i)
	if err != nil {
		return nil
	}
	return &v
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrStrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
