package storage

import (
	"fmt"
	"time"

	badger "github.com/dgraph-io/badger/v3"
)

// BadgerDB implements KeyValue and represents the application's connection
// to BadgerDB.
type BadgerDB struct {
	connection *badger.DB
	keyTTL     time.Duration // TTL for each key in the db
}

// NewBadgerDB initializes the BadgerDB embedded database. It is up to the
// caller to close the database with Close().
func NewBadgerDB(conf *KVConfig) (*BadgerDB, error) {
	// Open the Badger database at dirPath.
	// See: https://dgraph.io/docs/badger/get-started/#opening-a-database
	db, err := badger.Open(badger.DefaultOptions(conf.StorageDirPath))

	if err != nil {
		return &BadgerDB{}, fmt.Errorf("can't open the db connection: %v", err)
	}

	return &BadgerDB{
		connection: db,
		keyTTL:     conf.KeyTTLDuration,
	}, nil
}

// Put upserts an entry
func (db *BadgerDB) Put(entry KVEntry) error {
	err := db.connection.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(entry.Key, entry.Value).WithTTL(db.keyTTL)
		err := txn.SetEntry(e)
		if err != nil {
			return fmt.Errorf("could not set the KV pair: %v", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %v", err)
	}
	return nil
}

// Read returns an entry by key.
func (db *BadgerDB) Read(key []byte) (KVEntry, error) {
	var val []byte
	// See: https://dgraph.io/docs/badger/get-started/#read-only-transactions
	err := db.connection.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)

		if err != nil {
			return fmt.Errorf("can't retrieve a value for the key provided: %v", err)
		}

		// We copy values rather than return them directly because item.Value()
		// is considered undefined behavior outside a transaction.
		// https://godoc.org/github.com/dgraph-io/badger#Item.Value
		_, err = item.ValueCopy(val)

		if err != nil {
			return fmt.Errorf("can't copy the value from the database: %v", err)
		}
		return nil
	})
	if err != nil {
		return KVEntry{}, err
	}
	return KVEntry{
		Key:   key,
		Value: val,
	}, nil
}

// Cleanup performs BadgerDB's garbage collection routine with the
// recommended discardRatio.
//
// See: https://pkg.go.dev/github.com/ipsn/go-ipfs/gxlibs/github.com/dgraph-io/badger#DB.RunValueLogGC
//
// This is the only time old records are actually removed, so make sure you're
// setting TTLs for records!
func (db *BadgerDB) Cleanup() error {
	var discardRatio float64 = .5
	err := db.connection.RunValueLogGC(discardRatio)
	// If the GC determines that it can't rewrite anything, don't worry the
	// caller--just skip it
	if err.Error() == badger.ErrNoRewrite.Error() {
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

// Close tears down the database connection. You should defer this.
func (db *BadgerDB) Close() {
	err := db.connection.Close()
	if err != nil {
		panic(fmt.Sprintf("could not close the database: %v", err))
	}
}
