package persistent

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNoRows_Success(t *testing.T) {
	t.Parallel()
	assert.True(t, isNoRows(pgx.ErrNoRows))
}

func TestIsNoRows_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, isNoRows(nil))
	assert.False(t, isNoRows(assert.AnError))
}

func TestIsPgUniqueViolation_Success(t *testing.T) {
	t.Parallel()
	err := &pgconn.PgError{Code: "23505"}
	assert.True(t, isPgUniqueViolation(err))
}

func TestIsPgUniqueViolation_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, isPgUniqueViolation(nil))
	assert.False(t, isPgUniqueViolation(&pgconn.PgError{Code: "23503"}))
}

func TestPtrTimeToTime_Success(t *testing.T) {
	t.Parallel()
	ts := time.Now()
	got := ptrTimeToTime(&ts)
	assert.Equal(t, ts, got)
}

func TestPtrTimeToTime_Error(t *testing.T) {
	t.Parallel()
	got := ptrTimeToTime(nil)
	assert.True(t, got.IsZero())
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

func TestInt32PtrToInt_Success(t *testing.T) {
	t.Parallel()
	v := int32(42)
	assert.Equal(t, 42, int32PtrToInt(&v))
}

func TestInt32PtrToInt_Error(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, int32PtrToInt(nil))
}

func TestBoolPtrToBool_Success(t *testing.T) {
	t.Parallel()
	v := true
	assert.True(t, boolPtrToBool(&v))
}

func TestBoolPtrToBool_Error(t *testing.T) {
	t.Parallel()
	assert.False(t, boolPtrToBool(nil))
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

func TestStrPtrOrNil_Success(t *testing.T) {
	t.Parallel()
	got := strPtrOrNil("x")
	assert.NotNil(t, got)
	assert.Equal(t, "x", *got)
}

func TestStrPtrOrNil_Error(t *testing.T) {
	t.Parallel()
	got := strPtrOrNil("")
	assert.Nil(t, got)
}

func TestPtrStrToStr_Success(t *testing.T) {
	t.Parallel()
	s := "hello"
	assert.Equal(t, "hello", ptrStrToStr(&s))
}

func TestPtrStrToStr_Error(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", ptrStrToStr(nil))
}
