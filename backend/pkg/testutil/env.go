package testutil

import "os"

func GetEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}

	return fallback
}
