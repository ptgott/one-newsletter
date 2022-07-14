package storage

import (
	"time"

	"github.com/ptgott/one-newsletter/userconfig"
)

// SetUpDB initializes a KeyValue using the provided configuration and scraping
// interval. If the path is blank, this will return a NoOpDB.
func SetUpDB(c userconfig.Meta) (KeyValue, error) {

	var db KeyValue
	if c.Scraping.NoEmail || c.Scraping.OneOff {
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
		time.Duration(2)*c.Scraping.Interval,
	)
	if err != nil {
		return nil, err
	}
	return db, nil

}
