package storage

import "time"

// KVConfig contains settings specific to BadgerDB connections
type KVConfig struct {
	StorageDirPath  string        `yaml:"storageDir" json:"storageDir"`
	KeyTTLDuration  time.Duration `yaml:"keyTTL" json:"keyTTL"`
	CleanupInterval time.Duration `yaml:"cleanupInterval" json:"CleanupInterval"`
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
