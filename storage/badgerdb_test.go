package storage

import (
	"reflect"
	"testing"
	"time"
)

// We test all BadgerDB read/write utility functions here for a simple case. While
// other projects define test-specific utility functions for, e.g., opening
// a BadgerDB connection (e.g., Jaeger [1]), all DB operations are wrapped
// in a helper for use by the application. We'll use these helpers, rather than
// ones defined just for tests.
//
// [1]: https://github.com/jaegertracing/jaeger/blob/740264bd4c7a7cca27f0eb47d80cd8f8fcbd5906/plugin/storage/badger/spanstore/cache_test.go#L109-L126
func TestSimpleBadgerDBReadWrite(t *testing.T) {
	dir := t.TempDir()
	conf := KVConfig{
		StorageDirPath: dir,
		// Set these durations to a very long value since we don't expect
		// keys to be cleaned up during the test
		KeyTTLDuration: time.Duration(10) * time.Second,
	}
	db, err := NewBadgerDB(&conf)

	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	kv := KVEntry{
		Key:   []byte("Hello"),
		Value: []byte("World"),
	}

	err = db.Put(kv)

	if err != nil {
		t.Fatal(err)
	}

	kv2, err := db.Read(kv.Key)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(kv, kv2) {
		t.Fatal("newly created and newly read KV entries do not match")
	}

}
