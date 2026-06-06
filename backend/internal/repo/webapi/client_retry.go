package webapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const oauthRetryMaxElapsed = 30 * time.Second

// doWithRetry executes an HTTP request with exponential-backoff retry for
// transient failures. mkReq is called on each attempt to build a fresh request.
// Network errors and HTTP 5xx responses are retried; non-5xx responses are
// returned as-is. Retries stop after oauthRetryMaxElapsed or when ctx is done.
func doWithRetry(ctx context.Context, client *http.Client, mkReq func() (*http.Request, error)) (*http.Response, error) {
	var resp *http.Response

	op := func() error {
		req, err := mkReq()
		if err != nil {
			return err
		}

		r, err := client.Do(req) // #nosec G704 -- req URL from OAuth provider config, not user input
		if err != nil {
			return err
		}

		if r.StatusCode >= http.StatusInternalServerError {
			_ = r.Body.Close()

			return fmt.Errorf("OAuthClient - doWithRetry - bad status: oauth API returned %d", r.StatusCode)
		}

		resp = r

		return nil
	}
	bo := backoff.NewExponentialBackOff()

	bo.MaxElapsedTime = oauthRetryMaxElapsed

	err := backoff.Retry(op, backoff.WithContext(bo, ctx))
	if err != nil {
		return nil, err
	}

	return resp, nil
}
