package storage

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
	// Remove a Set permanently from storage
	Delete(key []byte) error
	// Drain/tear down the connection, or something analogous for
	// an embedded database
	Close() error
}

// KVEntry is what we'll write to and read from the KV store
type KVEntry struct {
	Key   []byte
	Value []byte
}
