package webapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const oauthRetryMaxElapsed = 30 * time.Second

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
		if r.StatusCode >= 500 {
			_ = r.Body.Close()
			return fmt.Errorf("oauth API returned %d", r.StatusCode)
		}
		resp = r
		return nil
	}
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = oauthRetryMaxElapsed
	if err := backoff.Retry(op, backoff.WithContext(bo, ctx)); err != nil {
		return nil, err
	}
	return resp, nil
}
