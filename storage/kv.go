package storage

import (
	"errors"
	"fmt"
	"time"
)

// KVConfig contains settings specific to BadgerDB connections
type KVConfig struct {
	StorageDirPath  string        `yaml:"storageDir"`
	KeyTTLDuration  time.Duration `yaml:"keyTTL"`
	CleanupInterval time.Duration `yaml:"cleanupInterval"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// https://pkg.go.dev/gopkg.in/yaml.v2#Unmarshaler
// It unmarshals a Config from YAML. We need to do this to grab some
// time.Durations from user-provided strings.
func (c *KVConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return fmt.Errorf("can't parse the storage config: %v", err)
	}

	sp, ok := v["storageDir"]
	if !ok {
		return errors.New(
			"user-provided storage config does not include a storage path",
		)
	}
	c.StorageDirPath = sp

	d, ok := v["keyTTL"]
	if !ok {
		return errors.New(
			"user-provided storage config does not include a key TTL",
		)
	}
	pd, err := time.ParseDuration(d)
	if err != nil {
		return fmt.Errorf(
			"can't parse the user-provided key TTL interval as a duration: %v",
			err,
		)
	}
	c.KeyTTLDuration = pd

	ci, ok := v["cleanupInterval"]
	if !ok {
		return errors.New(
			"user-provided storage config does not include a cleanup interval",
		)
	}
	cd, err := time.ParseDuration(ci)
	if err != nil {
		return fmt.Errorf(
			"can't parse the user-provided key cleanup interval as a duration: %v",
			err,
		)
	}
	c.CleanupInterval = cd

	return nil
}

// KeyValue exposes a common interface for performing CRUD operations on an
// underlying storage layer. Assumes some kind of persistent KV store
// for linksrc.Sets.
//
// Implentations need to include connection logic in code to initialize
// a Store.
type KeyValue interface {
	// Replace the value of a Set or create a new one if it doesn't exist
	Put(KVEntry) error
	// Return a Set given its key
	Read(key []byte) (KVEntry, error)
	// Cleanup performs routine deletion of old records. We assign
	// TTLs to KV pairs and delete them periodically.
	Cleanup() error
	// Drain/tear down the connection, or something analogous for
	// an embedded database
	Close() error
}

// KVEntry is what we'll write to and read from the KV store
type KVEntry struct {
	Key   []byte
	Value []byte
}
