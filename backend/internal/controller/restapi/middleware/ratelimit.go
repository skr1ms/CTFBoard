package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mhttp "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimit(client *redis.Client, keyPrefix string, limit int64, window time.Duration, keyFunc func(r *http.Request) (string, error)) func(next http.Handler) http.Handler {
	store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix:   "limiter:" + keyPrefix,
		MaxRetry: 3,
	})
	if err != nil {
		log.Fatal(err)
	}

	rate := limiter.Rate{
		Period: window,
		Limit:  limit,
	}
	instance := limiter.New(store, rate)

	middleware := mhttp.NewMiddleware(instance, mhttp.WithKeyGetter(func(r *http.Request) string {
		key, err := keyFunc(r)
		if err != nil {
			return ""
		}
		return key
	}))

	return middleware.Handler
}

func CheckRateLimit(ctx context.Context, client *redis.Client, keyPrefix, keySuffix string, limit int64, window time.Duration) (bool, error) {
	store, err := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: "limiter:" + keyPrefix,
	})
	if err != nil {
		return false, err
	}

	rate := limiter.Rate{
		Period: window,
		Limit:  limit,
	}

	instance := limiter.New(store, rate)

	context, err := instance.Get(ctx, keySuffix)
	if err != nil {
		return false, err
	}

	return !context.Reached, nil
}
