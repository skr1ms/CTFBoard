package redis_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/stretchr/testify/assert"
	testRedis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestRedis_Incr_Success(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := testRedis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Skipf("Skipping integration test due to container failure: %v", err)
		return
	}
	defer func() { _ = redisContainer.Terminate(ctx) }()

	connStr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// connStr is redis://host:port
	u, err := url.Parse(connStr)
	if err != nil {
		t.Fatal(err)
	}

	host := u.Hostname()
	port := u.Port()

	client, err := redis.New(host, port, "")
	assert.NoError(t, err)
	defer func() { _ = client.Close() }()

	// Test Incr
	cmd := client.Incr(ctx, "test_counter")
	assert.NoError(t, cmd.Err())
	assert.Equal(t, int64(1), cmd.Val())

	cmd2 := client.Incr(ctx, "test_counter")
	assert.NoError(t, cmd2.Err())
	assert.Equal(t, int64(2), cmd2.Val())
}

func TestRedis_Expire_Success(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := testRedis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Skipf("Skipping integration test due to container failure: %v", err)
		return
	}
	defer func() { _ = redisContainer.Terminate(ctx) }()

	connStr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatal(err)
	}

	u, err := url.Parse(connStr)
	if err != nil {
		t.Fatal(err)
	}

	client, err := redis.New(u.Hostname(), u.Port(), "")
	assert.NoError(t, err)
	defer func() { _ = client.Close() }()

	// Setup key
	client.Incr(ctx, "expire_key")

	// Test Expire
	boolCmd := client.Expire(ctx, "expire_key", time.Hour)
	assert.NoError(t, boolCmd.Err())
	assert.True(t, boolCmd.Val())
}

func TestRedis_Incr_Error(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := testRedis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Skipf("Skipping integration test due to container failure: %v", err)
		return
	}
	// Do NOT defer terminate immediately, we will terminate manually to simulate failure

	connStr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatal(err)
	}

	u, err := url.Parse(connStr)
	if err != nil {
		t.Fatal(err)
	}

	client, err := redis.New(u.Hostname(), u.Port(), "")
	assert.NoError(t, err)
	defer func() { _ = client.Close() }()

	// Terminate container to cause error
	if err := redisContainer.Terminate(ctx); err != nil {
		t.Fatal(err)
	}

	// Retry until error occurs or timeout (Circuit Breaker logic might need multiple failures)
	assert.Eventually(t, func() bool {
		cmd := client.Incr(ctx, "fail_counter")
		return cmd.Err() != nil
	}, 10*time.Second, 100*time.Millisecond, "Expected Incr to eventually fail after Redis termination")
}
