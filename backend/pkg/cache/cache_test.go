package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	client, _ := redismock.NewClientMock()
	c := New(client)
	require.NotNil(t, c)
}

func TestGetOrLoad_CacheHit(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	key := "k"
	ttl := time.Minute

	mock.ExpectGet(key).SetVal(`42`)
	loadCalled := false
	got, err := GetOrLoad(c, ctx, key, ttl, func() (int, error) {
		loadCalled = true
		return 0, errors.New("should not run")
	})
	require.NoError(t, err)
	assert.Equal(t, 42, got)
	assert.False(t, loadCalled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrLoad_CacheMiss_StoresAndReturns(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	key := "k"
	ttl := time.Minute

	mock.ExpectGet(key).RedisNil()
	mock.ExpectSet(key, []byte(`42`), ttl).SetVal("OK")
	got, err := GetOrLoad(c, ctx, key, ttl, func() (int, error) {
		return 42, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrLoad_LoadError(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	key := "k"
	ttl := time.Minute
	loadErr := errors.New("load failed")

	mock.ExpectGet(key).RedisNil()
	got, err := GetOrLoad(c, ctx, key, ttl, func() (int, error) {
		return 0, loadErr
	})
	require.Error(t, err)
	assert.Equal(t, 0, got)
	assert.ErrorIs(t, err, loadErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrLoad_StructType(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	key := "user:1"
	ttl := time.Minute

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	mock.ExpectGet(key).RedisNil()
	mock.ExpectSet(key, []byte(`{"id":1,"name":"alice"}`), ttl).SetVal("OK")
	got, err := GetOrLoad(c, ctx, key, ttl, func() (User, error) {
		return User{ID: 1, Name: "alice"}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, User{ID: 1, Name: "alice"}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCache_Del_EmptyKeys(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	c.Del(ctx)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCache_Del_WithKeys(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	mock.ExpectDel("a", "b").SetVal(2)
	c.Del(ctx, "a", "b")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCache_Set_Success(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	key := "k"
	ttl := time.Minute
	mock.ExpectSet(key, []byte(`42`), ttl).SetVal("OK")
	err := c.Set(ctx, key, 42, ttl)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCache_Set_UnmarshalableValue(t *testing.T) {
	client, _ := redismock.NewClientMock()
	c := New(client)
	ctx := context.Background()
	err := c.Set(ctx, "k", make(chan int), time.Minute)
	require.Error(t, err)
}
