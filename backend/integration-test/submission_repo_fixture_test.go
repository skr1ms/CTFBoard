package integration_test

import (
	"time"
)

// timeNowMinus returns a time.Time that is `seconds` seconds before now.
func timeNowMinus(seconds int) time.Time {
	return time.Now().Add(-time.Duration(seconds) * time.Second)
}
