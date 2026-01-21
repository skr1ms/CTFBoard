package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	fmt.Printf("Config: %s not found in environment, using default: '%s'\n", key, defaultValue)
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			fmt.Printf("Config: %s has invalid integer value, using default: %d\n", key, defaultValue)
			return defaultValue
		}
		return intValue
	}
	fmt.Printf("Config: %s not found in environment, using default: %d\n", key, defaultValue)
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Printf("Config: %s has invalid boolean value, using default: %v\n", key, defaultValue)
			return defaultValue
		}
		return boolValue
	}
	fmt.Printf("Config: %s not found in environment, using default: %v\n", key, defaultValue)
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
