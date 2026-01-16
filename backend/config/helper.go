package config

import (
	"os"
	"strconv"
	"strings"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return intValue
	}
	return defaultValue
}

func parseCORSOrigins(s string) []string {
	if s == "" {
		return []string{}
	}
	origins := strings.Split(s, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return origins
}
