package helper

import "github.com/google/uuid"

// UID returns a short unique string for test data (e.g. usernames, emails) to avoid collisions when running tests in parallel.
func UID() string {
	return uuid.NewString()[:8]
}
