package poller

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Scrapes must take place at a minimum every 5s. We'll probably use a much
// larger interval for a daily newsletter, but 5s is a failsafe to make
// sure we're not accidentally DOSing our link sources.
const minDurationMS int64 = 5000 // using MS since it's an int not a float

// Config contains options for polling online publications for links
type Config struct {
	Interval time.Duration
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// https://pkg.go.dev/gopkg.in/yaml.v2#Unmarshaler
// It unmarshals a Config from YAML. We need to do this to grab the
// Interval from a user-provided string.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return fmt.Errorf("can't parse the polling config: %v", err)
	}

	d, ok := v["interval"]

	if !ok {
		return errors.New(
			"user-provided polling config does not include an interval",
		)
	}

	pd, err := time.ParseDuration(d)

	if err != nil {
		return fmt.Errorf(
			"can't parse the user-provided polling interval as a duration: %v",
			err,
		)
	}

	if pd.Milliseconds() == 0 {
		return errors.New("polling interval must be greater than zero")
	}

	if pd.Milliseconds() < minDurationMS {
		minDurS := minDurationMS / 1000
		return fmt.Errorf("polling interval must be at least %v seconds", minDurS)
	}

	c.Interval = pd

	return nil
}

// Client handles HTTP requests, including transient state, when polling
// publication websites
type Client struct {
	http.Client
}
