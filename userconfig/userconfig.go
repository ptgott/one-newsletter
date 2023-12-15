package userconfig

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/rs/zerolog/log"

	"github.com/ptgott/one-newsletter/email"

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
	// Run the scraper once, then exit
	OneOff bool
	// Print the HTML body of a single email to stdout and exit to help test
	// configuration.
	TestMode bool
	// Number of days we keep a link in the database before marking it
	// expired.
	LinkExpiryDays uint
}

// CheckAndSetDefaults validates s and either returns a copy of s with default
// settings applied or returns an error due to an invalid configuration
func (s *Scraping) CheckAndSetDefaults() (Scraping, error) {

	i := s.Interval.Milliseconds()
	if i == 0 {
		return Scraping{}, errors.New(
			"user-provided config does not include a polling interval",
		)
	}

	if i < minDurationMS {
		minDurS := minDurationMS / 1000
		return Scraping{}, fmt.Errorf("polling interval must be at least %v seconds", minDurS)
	}
	if s.StorageDirPath == "" {
		return Scraping{}, errors.New(
			"user-provided config does not include a storage path",
		)
	}
	if s.LinkExpiryDays == 0 {
		s.LinkExpiryDays = 180
	}

	return *s, nil
}

// UnmarshalYAML parses a user-provided YAML configuration, returning any
// parsing errors.
func (s *Scraping) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return fmt.Errorf("can't parse the user config: %v", err)
	}

	d, ok := v["interval"]

	if !ok {
		d = "0s"
	}

	pd, err := time.ParseDuration(d)

	if err != nil {
		return fmt.Errorf(
			"can't parse the user-provided polling interval as a duration: %v",
			err,
		)
	}

	s.Interval = pd

	sp, ok := v["storageDir"]
	if !ok {
		sp = ""
	}

	s.StorageDirPath = sp

	li, ok := v["linkExpiryDays"]
	if !ok {
		li = "0"
	}

	lid, err := strconv.Atoi(li)
	if err != nil {
		return fmt.Errorf("can't parse the link expiry as an integer")
	}
	s.LinkExpiryDays = uint(lid)

	return nil
}

// CheckAndSetDefaults validates m and either returns a copy of m with default
// settings applied or returns an error due to an invalid configuration
func (m *Meta) CheckAndSetDefaults() (Meta, error) {
	c := Meta{}

	s, err := m.Scraping.CheckAndSetDefaults()
	if err != nil {
		return Meta{}, err
	}
	c.Scraping = s

	e, err := m.EmailSettings.CheckAndSetDefaults()
	if err != nil {
		return Meta{}, err
	}
	c.EmailSettings = e

	c.LinkSources = make([]linksrc.Config, len(m.LinkSources))
	for n, s := range m.LinkSources {
		ns, err := s.CheckAndSetDefaults()
		if err != nil {
			return Meta{}, err
		}
		c.LinkSources[n] = ns
	}

	return c, nil

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

	// Since this is a one-off or a test, set the data directory to an
	// empty string to disable database operations.
	if m.Scraping.OneOff || m.Scraping.TestMode {
		m.Scraping.StorageDirPath = ""
		log.Debug().Msg(
			"disabling database operations",
		)
	}

	return &m, nil

}
