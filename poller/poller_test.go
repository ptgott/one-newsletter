package poller

import (
	"bytes"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		description   string
		shouldBeError bool
		input         string
	}{
		{
			description:   "valid case",
			shouldBeError: false,
			input:         `interval: 5s`,
		},
		{
			description:   "not an object",
			shouldBeError: true,
			input:         `[]`,
		},
		{
			description:   "no interval key",
			shouldBeError: true,
			input:         `cadence: 5s`,
		},
		{
			description:   "unparseable duration",
			shouldBeError: true,
			input:         `interval: 5y`,
		},
		{
			description:   "zero interval",
			shouldBeError: true,
			input:         `interval: 0s`,
		},
		{
			description:   "interval less than 5s",
			shouldBeError: true,
			input:         `interval: 100ms`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var c Config
			dec := yaml.NewDecoder(bytes.NewBuffer([]byte(tc.input)))
			if err := dec.Decode(&c); (err != nil) != tc.shouldBeError {
				t.Errorf(
					"expected error status of %v but got %v with error %v",
					tc.shouldBeError,
					err != nil,
					err,
				)
			}
		})
	}
}
