package poller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Scrapes must take place at a minimum every 10s. We'll probably use a much
// larger interval for a daily newsletter, but 10s is a failsafe to make
// sure we're not accidentally DOSing our link sources.
const minDurationNano int64 = 10e10

// Config contains options for polling online publications for links
type Config struct {
	Interval time.Duration
}

// Validate returns an error if the Config is invalid
func (c Config) Validate() error {
	if c.Interval.Nanoseconds() == 0 {
		return errors.New("polling interval must be greater than zero")
	}
	if c.Interval.Nanoseconds() <= minDurationNano {
		minDurS := minDurationNano / 10e9
		return fmt.Errorf("polling interval must be at least %v seconds", minDurS)
	}
	return nil
}

// Client handles HTTP requests, including transient state, when polling
// publication websites
type Client struct {
	http.Client
}

// Poll retrieves HTML from an HTTP endpoint for reading downstream
func (c Client) Poll(url string) (io.Reader, error) {
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}

	r := bytes.Buffer{}
	_, err = r.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return &r, nil
}
