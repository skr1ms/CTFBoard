package cache

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
)

const (
	subBackoffInitial = 1 * time.Second
	subBackoffMax     = 30 * time.Second
)

// SubscribeInvalidation runs a blocking loop that subscribes to a Redis PubSub channel
// and calls onMessage for each received event. It reconnects with exponential backoff
// (1s → 30s, unlimited retries) and exits cleanly when stopCtx is cancelled.
// Intended to run inside a goroutine - the caller is responsible for go SubscribeInvalidation(...).
func SubscribeInvalidation(
	stopCtx context.Context,
	pubsub cachekit.PubSubStore,
	channel string,
	onMessage func(),
	logger logkit.Logger,
	logPrefix string,
) {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = subBackoffInitial
	bo.MaxInterval = subBackoffMax
	bo.MaxElapsedTime = 0

	notify := func(err error, d time.Duration) {
		logger.WithError(err).Warn(logPrefix+": subscribe to invalidation channel failed, retrying",
			logkit.Fields{"backoff_sec": d.Seconds()})
	}

	for {
		select {
		case <-stopCtx.Done():
			return
		default:
		}

		var ch <-chan string

		op := func() error {
			var err error

			ch, err = pubsub.Subscribe(stopCtx, channel)

			return err
		}
		if err := backoff.RetryNotify(op, backoff.WithContext(bo, stopCtx), notify); err != nil {
			return
		}

		bo.Reset()

	readLoop:
		for {
			select {
			case <-stopCtx.Done():
				return
			case _, ok := <-ch:
				if !ok {
					break readLoop
				}

				onMessage()
			}
		}

		next := bo.NextBackOff()
		if next == backoff.Stop {
			return
		}

		logger.Warn(logPrefix+": invalidation subscriber stopped, reconnecting",
			logkit.Fields{"backoff_sec": next.Seconds()})

		t := time.NewTimer(next)
		select {
		case <-stopCtx.Done():
			t.Stop()

			return
		case <-t.C:
		}
	}
}
