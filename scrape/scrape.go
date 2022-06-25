package scrape

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ptgott/one-newsletter/html"
	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/ptgott/one-newsletter/storage"
	"github.com/ptgott/one-newsletter/userconfig"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// For time.Ticker ticks
	TickCh <-chan time.Time
	// Channel to send errors to
	ErrCh chan error
	// Channel for stopping the scraper
	StopCh chan struct{}
	// Writer for a message to display when a scrape has finished.The means
	// of display is controlled by the caller. Intended for email text shown
	// when the --noemail flag is used.
	OutputWr io.Writer
}

// Run conducts a single scrape and email cycle and returns the first error
// encountered. It reads the user config anew at the beginning of each cycle. At
// the end of a scrape cycle, it sends an email or, depending on the config,
// writes a plaintext version of the email message to outwr.
func Run(outwr io.Writer, config *userconfig.Meta) error {

	httpClient := http.Client{
		// Determined arbitrarily. We don't want to wait forever for a
		// request to complete, but the cadence of the newsletter means
		// that a minute of extra waiting is probably okay.
		Timeout: time.Duration(60) * time.Second,
	}

	var db storage.KeyValue
	if config.Scraping.NoEmail || config.Scraping.OneOff {
		db = &storage.NoOpDB{}
	} else {
		var err error
		db, err = storage.NewBadgerDB(
			config.Scraping.StorageDirPath,
			// A key inserted at one polling
			// interval expires two intervals
			// later, meaning that the interval
			// after a link is collected,
			// we can still compare it to newly
			// collected links.
			time.Duration(2)*config.Scraping.Interval,
		)
		if err != nil {
			return err
		}
	}

	log.Info().Msg("set up the database connection successfully")
	log.Info().
		Int("count", len(config.LinkSources)).
		Msg("launching scrapers")
	var wg sync.WaitGroup
	d := html.NewEmailData()

	// buffer the results of the latest scrape so we can perform a diff
	// with the previous scrape and build an email body
	emailBuildCh := make(chan linksrc.Set, len(config.LinkSources))
	wg.Add(len(config.LinkSources))
	var ec chan error
	for _, ls := range config.LinkSources {
		go func(
			lc linksrc.Config,
			g *sync.WaitGroup,
			bc chan linksrc.Set,
			ech chan error,
		) {
			defer g.Done()
			// Try the scrape request only once. If we get a non-2xx
			// response, it's probably not something we can expect to
			// clear up after retrying.
			r, err := httpClient.Get(lc.URL.String())
			if err != nil {
				ech <- err
				return
			}
			defer r.Body.Close()
			s := linksrc.NewSet(r.Body, lc, r.StatusCode)

			bc <- s

		}(ls, &wg, emailBuildCh, ec)
	}
	wg.Wait()

	// Return the first error sent to the channel
	select {
	case err := <-ec:
		return err
	default:
	}
	// TODO: Having the receiver close the channel is not how close()
	// was intended to be used, but senders have no way of knowing
	// when to close the channel, and we need to use close() in order
	// to range over the channel below.
	close(emailBuildCh)
	log.Info().
		Msg("done with one round of scraping")
	for set := range emailBuildCh {
		// See if any items are missing in the db. If so, store them
		// and add them to a new email body.
		for _, item := range set.LinkItems() {
			// Read returns a "key not found" error if a key is not found.
			// https://pkg.go.dev/github.com/dgraph-io/badger#Txn.Get
			_, err := db.Read(item.Key())
			// If the Item already exists in the database,
			if err == nil {
				set.RemoveLinkItem(item)
			} else {
				log.Info().Msg("storing a link item in the database")
				err = db.Put(item.NewKVEntry())
				if err != nil {
					log.Error().
						Err(err).
						Msg("error saving a link item")
					continue
				}
			}
		}
		d.Add(set)
		log.Info().
			Int("itemCount", set.CountLinkItems()).
			Str("setName", set.Name).
			Msg("added items to the email")
	}

	// Get rid of old keys just before we close
	err := db.Cleanup()
	if err != nil {
		log.Error().Err(err).Msg("error cleaning up the database")
	}
	// Close the connection here so BadgerDB can flush to disk.
	// Otherwise, BadgerDB has to reach its MaxTableSize before it
	// flushes--we want to write the results of each scraping round to
	// disk, and there's no need to keep the DB connection open while
	// waiting for the next scrape.
	//
	// https://pkg.go.dev/github.com/dgraph-io/badger#readme-i-don-t-see-any-disk-writes-why
	db.Close()
	log.Info().Msg("closed the database to flush data to disk")
	bod := d.GenerateBody()
	txt := d.GenerateText()
	log.Info().Msg("attempting to send an email")

	if config.Scraping.NoEmail {
		if outwr == nil {
			log.Warn().Msg(
				"a writer is unavailable for receiving the output message",
			)

		} else {
			if _, err := outwr.Write([]byte(bod)); err != nil {
				log.Error().Err(err).Msg("cannot write the message output")
			}
		}
	} else {
		err = config.EmailSettings.SendNewsletter([]byte(txt), []byte(bod))
		if err != nil {
			log.Error().Err(err).Msg("error sending an email")
		}
	}

	// We're only doing this once, so get out of the main loop
	if config.Scraping.OneOff {
		return nil
	}
	return nil

}

// StartLoop begins the main sequence of scraping websites for links every
// interval (defined by tc) with the provided config. If an s.ErrCh is provided,
// sends any errors to it. Send a struct{} to sc to stop the scraper.
func StartLoop(s *Config, c *userconfig.Meta) {

	// The first email will be sent after the scrape interval
	// TODO: This should send the first email immediately instead.
	if !c.Scraping.OneOff {
		<-s.TickCh
	}

	// Run the first scrape immediately
	err := Run(s.OutputWr, c)
	// Make a weak effort to send any errors encountered to the provided
	// error channel. If there are no receivers,
	if err != nil {
		select {
		case s.ErrCh <- err:
		default:
		}

	}

	// enter the main scraping/email sending loop
	for !c.Scraping.OneOff {
		select {
		case <-s.StopCh:
			return
		case <-s.TickCh:
			go func(io.Writer, *Config) {
				err := Run(s.OutputWr, c)
				if err != nil {
					select {
					case s.ErrCh <- err:
					default:
					}
				}
			}(s.OutputWr, s)
		}

	}
}
