package plugin

import (
	"fmt"
	"os"
)

func ValueOrFallback(value string, fallback func() (string, error)) (string, error) {
	if value != "" {
		return value, nil
	}

	return fallback()
}

func FallbackFromEnv(key string) func() (string, error) {
	return func() (string, error) {
		value, ok := os.LookupEnv(key)

		if !ok || value == "" {
			return "", fmt.Errorf("%q must not be empty", key)
		}

		return value, nil
	}
}
