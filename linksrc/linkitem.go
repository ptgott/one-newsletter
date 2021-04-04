package linksrc

import (
	"bytes"
	"crypto/sha256"
	"divnews/storage"
	"encoding/binary"
	"time"
)

// LinkItem represents data for a single link item found within a
// list of links
type LinkItem struct {
	// using a string here because we'll let the downstream context deal
	// with parsing URLs etc. This comes from a website so we can't really
	// trust it.
	LinkURL string
	Caption string
}

// Key returns the key to use for determining whether a LinkItem has already
// been stored within the database
func (li LinkItem) Key() []byte {
	// The key is the hash of the entire serialized LinkItem. This lets us quickly
	// determine whether a LinkItem already exists in storage.
	k := sha256.New()
	k.Write([]byte(li.Caption))
	k.Write([]byte(li.LinkURL))
	return k.Sum(nil)
}

// NewKVEntry prepares the LinkItem to be saved in the KV database. Keys are
// SHA256 hashes of the entire LinkItem. Values are timestamps in seconds since
// the Unix epoch. Usually we'll just be checking whether newly fetched
// LinkItems are already saved. Eventually we might want to use the timestamp.
func (li LinkItem) NewKVEntry() storage.KVEntry {

	var buf bytes.Buffer

	// Using little endian order arbitrarily--if this ends up mattering, feel
	// free to change.
	//
	// Suppressing errors since they only come from the Buffer's Write method [1],
	// which always returns a nil error [2].
	// [1]: https://github.com/golang/go/blob/d0d38f0f707e69965a5f5a637fa568c646899d39/src/encoding/binary/binary.go#L375
	// [2]: https://github.com/golang/go/blob/d0d38f0f707e69965a5f5a637fa568c646899d39/src/bytes/buffer.go#L165-L175
	binary.Write(&buf, binary.LittleEndian, time.Now().Unix())

	return storage.KVEntry{
		Key:   li.Key(),
		Value: buf.Bytes(),
	}

}
