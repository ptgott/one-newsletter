package userconfig

import (
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
	dec := yaml.NewDecoder(r)

	var m Meta
	err := dec.Decode(&m)
	if err != nil {
		return &Meta{}, fmt.Errorf("can't read the config file as YAML: %v", err)
	}

	return &m, nil

}
