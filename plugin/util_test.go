package plugin

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValueOrFallback(t *testing.T) {
	cases := []struct {
		name          string
		value         string
		fallback      func() (string, error)
		expectedError error
		expectedValue string
	}{
		{
			name:          "value is provided",
			value:         "value",
			fallback:      func() (string, error) { return "", nil },
			expectedError: nil,
			expectedValue: "value",
		},
		{
			name:          "value is not provided, fallback is",
			value:         "",
			fallback:      func() (string, error) { return "fallback", nil },
			expectedError: nil,
			expectedValue: "fallback",
		},
		{
			name:          "value is provided, fallback is also provided",
			value:         "value",
			fallback:      func() (string, error) { return "fallback", nil },
			expectedError: nil,
			expectedValue: "value",
		},
		{
			name:          "value is not provided, neither is fallback",
			value:         "",
			fallback:      func() (string, error) { return "", fmt.Errorf("not found") },
			expectedError: errors.New("not found"),
			expectedValue: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			value, err := ValueOrFallback(c.value, c.fallback)

			assert.Equal(t, c.expectedValue, value, c.name)
			assert.Equal(t, c.expectedError, err, c.name)
		})
	}
}

func Test_FallbackFromEnv(t *testing.T) {
	cases := []struct {
		name          string
		value         string
		expectedError error
		expectedValue string
	}{
		{
			name:          "variable is set",
			value:         "value",
			expectedError: nil,
			expectedValue: "value",
		},
		{
			name:          "variable is not set",
			value:         "",
			expectedError: errors.New("\"TEST_ENV_VARIABLE\" must not be empty"),
			expectedValue: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fallback := FallbackFromEnv("TEST_ENV_VARIABLE")

			os.Setenv("TEST_ENV_VARIABLE", c.value)
			defer os.Unsetenv("TEST_ENV_VARIABLE")

			value, err := fallback()
			assert.Equal(t, c.expectedValue, value, c.name)
			assert.Equal(t, c.expectedError, err, c.name)
		})
	}
}
