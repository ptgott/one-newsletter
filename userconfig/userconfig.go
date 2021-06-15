package userconfig

import (
	"divnews/email"
	"divnews/linksrc"
	"errors"
	"fmt"
	"io"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Scrapes must take place at a minimum every 5s. We'll probably use a much
// larger interval for a daily newsletter, but 5s is a failsafe to make
// sure we're not accidentally DOSing our link sources.
const minDurationMS int64 = 5000 // using MS since it's an int not a float

// Meta represents all current config options that the application can use,
// i.e., after validation and parsing
type Meta struct {
	Scraping      Scraping         `yaml:"scraping"`
	EmailSettings email.UserConfig `yaml:"email"`
	LinkSources   []linksrc.Config `yaml:"link_sources"`
}

// Scraping contains config options that apply to One Newsletter's scraping
// behavior
type Scraping struct {
	Interval       time.Duration
	StorageDirPath string
}

// UnmarshalYAML implements yaml.Unmarshaler. It validates and parses
// user config for general scraping behavior
func (s *Scraping) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return fmt.Errorf("can't parse the user config: %v", err)
	}

	d, ok := v["interval"]

	if !ok {
		return errors.New(
			"user-provided config does not include a polling interval",
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

	s.Interval = pd

	sp, ok := v["storageDir"]
	if !ok {
		return errors.New(
			"user-provided config does not include a storage path",
		)
	}

	s.StorageDirPath = sp

	return nil
}

// Parse generates usable configurations from possibly arbitrary user input.
// An error indicates a problem with parsing or validation. The Reader r
// can be either JSON or YAML.
func Parse(r io.Reader) (*Meta, error) {
	var m Meta
	err := yaml.NewDecoder(r).Decode(&m)
	if err != nil {
		return &Meta{}, fmt.Errorf("can't read the config file as YAML: %v", err)
	}

	var es email.UserConfig = email.UserConfig{}
	if m.EmailSettings == es {
		return &Meta{}, errors.New("must include an \"email\" section")
	}

	var sc Scraping = Scraping{}
	if m.Scraping == sc {
		return &Meta{}, errors.New("must include a \"scraping\" section")
	}

	if len(m.LinkSources) == 0 {
		return &Meta{}, errors.New("must include at least one item within \"link_sources\"")
	}

	return &m, nil

}
