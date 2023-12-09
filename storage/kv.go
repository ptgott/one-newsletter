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
	// Cleanup performs routine deletion of old records. We assign
	// TTLs to KV pairs and delete them periodically.
	Cleanup() error
	// Drain/tear down the connection, or something analogous for an
	// embedded database. Implementations should handle retries or drain
	// connections internally and panic on failure, since there is nothing
	// the program can do if it can't close the database.
	Close()
}

// KVEntry is what we'll write to and read from the KV store
type KVEntry struct {
	Key   []byte
	Value []byte
}
