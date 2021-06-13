package storage

// storage contains the KeyValue interface for working with a persistent key/
// value store, as well as an implementation for BadgerDB. Note that the
// storage package isn't designed to represent _what_ is stored in the
// database, and deals only in opaque binary data.
