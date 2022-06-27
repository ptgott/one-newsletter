package storage

import (
	"time"
)

// SetUpDB initializes a KeyValue using the provided data directory path and
// scraping interval. If the path is blank, this will return a NoOpDB.
func SetUpDB(path string, interval time.Duration) (KeyValue, error) {

	var db KeyValue
	if path == "" {
		return &NoOpDB{}, nil
	}
	db, err := NewBadgerDB(
		path,
		// A key inserted at one polling
		// interval expires two intervals
		// later, meaning that the interval
		// after a link is collected,
		// we can still compare it to newly
		// collected links.
		time.Duration(2)*interval,
	)
	if err != nil {
		return nil, err
	}
	return db, nil

}
