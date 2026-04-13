package testutil

import "os"

// GetEnv returns the value of the environment variable named key, or fallback if the variable is unset.
func GetEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}
