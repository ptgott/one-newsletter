package storage

import "errors"

// NoOpDB is used when we need to avoid touching the storage layer while still
// preserving our interactions with an abstract database. The strategy is to
// return whatever value will prevent the calling context from further
// interacting with the storage layer.
//
// For get and put operations, we always return an error, so the caller knows that
// no actual data has been read or written.
//
// For database-wide operations, such as cleaning up or closing the database,
// we always return a nil error. This is because, since there is nothing to
// close or clean up, the operation is always successful.
type NoOpDB struct{}

// Put always returns an error so callers don't assume a new key has been
// written.
func (n *NoOpDB) Put(KVEntry) error {
	return errors.New("unable to write to the no-op database")
}

// Read always returns an error so callers don't assume a key has been read.
func (n *NoOpDB) Read(key []byte) (KVEntry, error) {
	return KVEntry{}, errors.New("entry not found in the no-op database")
}

// Cleanup always returns nil in order to prevent retries or panics, since we
// want to keep the program humming along without touching the storage layer.
func (n *NoOpDB) Cleanup() error {
	return nil
}

// Close is no-op
func (n *NoOpDB) Close() {
	return
}
