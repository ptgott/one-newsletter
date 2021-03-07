package userconfig

import (
	"bytes"
	"divnews/email"
	"divnews/linksrc"
	"divnews/poller"
	"divnews/storage"
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v2"
)

// Meta represents all current config options that the application can use,
// i.e., after validation and parsing
type Meta struct {
	EmailSettings   *email.UserConfig `yaml:"email"`
	LinkSources     []linksrc.Config  `yaml:"link_sources"`
	PollSettings    *poller.Config    `yaml:"polling"`
	StorageSettings *storage.KVConfig `yaml:"storage"`
}

// Parse generates usable configurations from possibly arbitrary user input.
// An error indicates a problem with parsing or validation. The Reader r
// can be either JSON or YAML.
func Parse(r io.Reader) (*Meta, error) {
	m, err := generateUntrusted(r)
	if err != nil {
		return &Meta{}, fmt.Errorf("can't parse the provided input into a configuration: %v", err)
	}

	err = validate(m)
	if err != nil {
		return &Meta{}, fmt.Errorf("invalid user configuration: %v", err)
	}

	return m, nil

}

// generateUntrusted produces a configuration from arbitrary input. Doesn't
// care about validation, so don't use the results of this within the
// application.
//
// The Reader r can be either JSON or YAML.
func generateUntrusted(r io.Reader) (*Meta, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return &Meta{}, fmt.Errorf("couldn't read from the provided config: %v", err)
	}

	var m Meta

	err = yaml.Unmarshal(buf.Bytes(), &m)

	if err != nil {
		return &Meta{}, fmt.Errorf("can't unmarshal the provided YAML: %v", err)
	}

	return &m, nil
}

// validate validates a Meta. We parse a Meta before validating so any
// parsing errors are picked up beforehand. An error indicates an invalid
// config
func validate(m *Meta) error {
	err := m.EmailSettings.Validate()
	if err != nil {
		return fmt.Errorf("invalid email settings: %v", err)
	}

	for _, ls := range m.LinkSources {
		if err = ls.Validate(); err != nil {
			return fmt.Errorf("invalid link source config: %v", err)
		}
	}

	err = m.PollSettings.Validate()
	if err != nil {
		return fmt.Errorf("invalid settings for the website poller: %v", err)
	}

	return nil

}
