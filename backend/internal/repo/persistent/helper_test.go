package persistent

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wahrwelt-kit/go-pgkit/pgutil"
)

func TestIsNoRows_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, pgutil.IsNoRows(pgx.ErrNoRows))
}

func TestIsNoRows_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, pgutil.IsNoRows(nil))
	assert.False(t, pgutil.IsNoRows(assert.AnError))
}

func TestIsPgUniqueViolation_Success(t *testing.T) {
	t.Parallel()
	err := &pgconn.PgError{Code: "23505"}
	assert.True(t, pgutil.IsPgUniqueViolation(err))
}

func TestIsPgUniqueViolation_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, pgutil.IsPgUniqueViolation(nil))
	assert.False(t, pgutil.IsPgUniqueViolation(&pgconn.PgError{Code: "23503"}))
}

func TestPtrTimeToTime_Success(t *testing.T) {
	t.Parallel()
	ts := time.Now()
	got := pgutil.PtrTimeToTime(&ts)
	assert.Equal(t, ts, got)
}

func TestPtrTimeToTime_Error(t *testing.T) {
	t.Parallel()
	got := pgutil.PtrTimeToTime(nil)
	assert.True(t, got.IsZero())
}

func TestTimestamptzToTime_Valid(t *testing.T) {
	t.Parallel()
	ts := time.Date(2025, 3, 6, 12, 0, 0, 0, time.UTC)
	in := pgtype.Timestamptz{Time: ts, Valid: true}
	got := pgutil.TimestamptzToTime(in)
	require.NotNil(t, got)
	assert.Equal(t, ts, *got)
}

func TestTimestamptzToTime_Invalid(t *testing.T) {
	t.Parallel()
	got := pgutil.TimestamptzToTime(pgtype.Timestamptz{})
	assert.Nil(t, got)
}

func TestTimeToTimestamptz_Valid(t *testing.T) {
	t.Parallel()
	ts := time.Date(2025, 3, 6, 12, 0, 0, 0, time.UTC)
	got := pgutil.TimeToTimestamptz(&ts)
	assert.True(t, got.Valid)
	assert.Equal(t, ts, got.Time)
}

func TestTimeToTimestamptz_Nil(t *testing.T) {
	t.Parallel()
	got := pgutil.TimeToTimestamptz(nil)
	assert.False(t, got.Valid)
	assert.True(t, got.Time.IsZero())
}

func TestTimestamptzToTime_TimeToTimestamptz_Roundtrip(t *testing.T) {
	t.Parallel()
	ts := time.Date(2025, 3, 6, 12, 0, 0, 0, time.UTC)
	pg := pgutil.TimeToTimestamptz(&ts)
	back := pgutil.TimestamptzToTime(pg)
	require.NotNil(t, back)
	assert.Equal(t, ts, *back)
}

func TestTimeFromNullableAny_Success(t *testing.T) {
	t.Parallel()
	ts := time.Now()
	assert.Equal(t, ts, timeFromNullableAny(ts))
	assert.Equal(t, ts, timeFromNullableAny(&ts))
}

func TestTimeFromNullableAny_Error(t *testing.T) {
	t.Parallel()
	got := timeFromNullableAny(nil)
	assert.True(t, got.IsZero())
	got = timeFromNullableAny("not time")
	assert.True(t, got.IsZero())
}

func TestConvertIntFieldsToInt32_Success(t *testing.T) {
	t.Parallel()
	fields := []IntField{{"A", 1}, {"B", 2}, {"C", 100}}
	got, err := ConvertIntFieldsToInt32(fields)
	assert.NoError(t, err)
	assert.Equal(t, []int32{1, 2, 100}, got)
}

func TestConvertIntFieldsToInt32_Error(t *testing.T) {
	t.Parallel()
	fields := []IntField{{"A", 1}, {"Bad", 1 << 31}}
	_, err := ConvertIntFieldsToInt32(fields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bad")
}

func TestIntToInt32Safe_Success(t *testing.T) {
	t.Parallel()
	got, err := intToInt32Safe(100)
	assert.NoError(t, err)
	assert.Equal(t, int32(100), got)
}

func TestIntToInt32Safe_Error(t *testing.T) {
	t.Parallel()
	_, err := intToInt32Safe(1 << 31)
	assert.Error(t, err)
	_, err = intToInt32Safe(-1<<31 - 1)
	assert.Error(t, err)
}

func TestIntToInt32Ptr_Success(t *testing.T) {
	t.Parallel()
	got, err := intToInt32Ptr(10)
	assert.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int32(10), *got)
}

func TestIntToInt32Ptr_Zero(t *testing.T) {
	t.Parallel()
	got, err := intToInt32Ptr(0)
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestIntToInt32Ptr_OutOfRange(t *testing.T) {
	t.Parallel()
	_, err := intToInt32Ptr(1 << 31)
	assert.Error(t, err)
	_, err = intToInt32Ptr(-1<<31 - 1)
	assert.Error(t, err)
}
