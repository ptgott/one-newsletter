package storage

import (
	"fmt"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// BadgerDB implements KeyValue and represents the application's connection
// to BadgerDB.
type BadgerDB struct {
	connection *badger.DB
	keyTTL     time.Duration // TTL for each key in the db
}

// badgerLogger lets us implement BadgerDB's Logger interface so we can log
// database events.
type badgerLogger struct {
	zerolog.Logger
}

// Debugf lets badgerLogger implement the BadgerDB Logger interface
func (bl badgerLogger) Debugf(s string, o ...interface{}) {
	bl.Logger.Debug().Msg(fmt.Sprintf(s, o...))
}

// Errorf lets badgerLogger implement the BadgerDB Logger interface
func (bl badgerLogger) Errorf(s string, o ...interface{}) {
	bl.Logger.Error().Msg(fmt.Sprintf(s, o...))
}

// Infof lets badgerLogger implement the BadgerDB Logger interface
func (bl badgerLogger) Infof(s string, o ...interface{}) {
	bl.Logger.Info().Msg(fmt.Sprintf(s, o...))
}

// Warningf lets badgerLogger implement the BadgerDB Logger interface
func (bl badgerLogger) Warningf(s string, o ...interface{}) {
	bl.Logger.Info().Msg(fmt.Sprintf(s, o...))
}

// NewBadgerDB initializes the BadgerDB embedded database given the provided
// storage directory path sd and TTL for keys. It is up to the caller to close
// the database with Close().
func NewBadgerDB(sd string, ttl time.Duration) (*BadgerDB, error) {
	// Open the Badger database at dirPath.
	// See: https://dgraph.io/docs/badger/get-started/#opening-a-database
	db, err := badger.Open(
		badger.DefaultOptions(sd).
			WithLogger(badgerLogger{log.Logger}).
			// Among other things, compacting on close updates discard info so
			// we can run value log GC later. Without this, the size of the data
			// directory will increase each polling interval.
			// https://github.com/dgraph-io/badger/blob/ca80206d2c0c869560d5b9cfdcab0307c807a54c/levels.go#L861
			WithCompactL0OnClose(true),
	)

	if err != nil {
		return &BadgerDB{}, fmt.Errorf("can't open the db connection: %v", err)
	}

	return &BadgerDB{
		connection: db,
		keyTTL:     ttl,
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
	// Based on:
	// https://dgraph.io/docs/badger/get-started/#using-key-value-pairs/
	var val []byte
	err := db.connection.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)

		if err != nil {
			return fmt.Errorf("can't retrieve a value for the key provided: %v", err)
		}

		err = item.Value(func(v []byte) error {
			// allocate a copy of v, rather than assign directly to v
			val = append([]byte{}, v...)
			return nil
		})

		if err != nil {
			return fmt.Errorf("can't retrieve the value from the database: %v", err)
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
	var err error
	// BadgerDB recommends running RunValueLogGC repeatedly since it only
	// removes one file at a time.
	for err = db.connection.RunValueLogGC(discardRatio); err == nil; {
		continue
	}
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
