package scrape

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/ptgott/one-newsletter/email"
	"github.com/ptgott/one-newsletter/html"
	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/ptgott/one-newsletter/storage"
	"github.com/ptgott/one-newsletter/userconfig"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// For time.Ticker ticks
	TickCh <-chan time.Time
	// Writer for a message to display when a scrape has finished.The means
	// of display is controlled by the caller. Intended for email text shown
	// when the --noemail flag is used.
	OutputWr      io.Writer
	ScheduleStore *userconfig.ScheduleStore
}

// Run conducts a single scrape and email cycle and returns the first error
// encountered. It reads the user config anew at the beginning of each cycle. At
// the end of a scrape cycle, it sends an email or, depending on the config,
// writes a plaintext version of the email message to outwr.
func Run(outwr io.Writer, scraping userconfig.Scraping, emailSettings email.UserConfig, newsletter userconfig.Newsletter) error {
	httpClient := http.Client{
		// Determined arbitrarily. We don't want to wait forever for a
		// request to complete, but the cadence of the newsletter means
		// that a minute of extra waiting is probably okay.
		Timeout: time.Duration(60) * time.Second,
	}

	var db storage.KeyValue
	if scraping.TestMode || scraping.OneOff {
		db = &storage.NoOpDB{}
	} else {
		var err error
		db, err = storage.NewBadgerDB(
			scraping.StorageDirPath,
			time.Duration(scraping.LinkExpiryDays*24)*time.Hour,
		)
		if err != nil {
			return err
		}
	}

	log.Info().Msg("set up the database connection successfully")
	log.Info().
		Msg("launching scrapers")
	var wg sync.WaitGroup
	d := html.NewNewsletterEmailData()

	// buffer the results of the latest scrape so we can perform a diff
	// with the previous scrape and build an email body
	emailBuildCh := make(chan linksrc.Set, len(newsletter.LinkSources))
	wg.Add(len(newsletter.LinkSources))
	var ec chan error
	for _, ls := range newsletter.LinkSources {
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
			ctx, cancel := context.WithTimeout(
				context.Background(),
				time.Duration(1)*time.Minute,
			)
			defer cancel()
			s := linksrc.NewSet(ctx, r.Body, lc, r.StatusCode)

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

	if scraping.TestMode {
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
		err = emailSettings.SendNewsletter([]byte(txt), []byte(bod))
		if err != nil {
			log.Error().Err(err).Msg("error sending an email")
		}
	}

	return nil
}

// StartLoop begins the main sequence of scraping websites for links every
// interval as specified in the provided config. If an s.ErrCh is provided,
// sends any errors to it. Send a struct{} to sc to stop the scraper.
func StartLoop(s *Config, c *userconfig.Meta) error {
	// Only running the loop once for the specified newsletter
	if c.Scraping.OneOff || c.Scraping.TestMode {
		n, ok := c.Newsletters[c.Scraping.NewsletterName]
		if !ok {
			return fmt.Errorf("cannot find a configuration for a newsletter named %q", c.Scraping.NewsletterName)
		}
		err := Run(s.OutputWr, c.Scraping, c.EmailSettings, n)
		if err != nil {
			return err
		}

		return nil
	}

	// TODO: Make the initial newsletter a confirmation that lists
	// information about all the newsletters we plan to send. Change the Run
	// call commented out below to reflect this. Once this is done, update
	// e2e tests so expectedEmails and the fakeTickChan arguments are
	// correct.
	//
	// 	// Run the first scrape immediately
	// 	err := Run(s.OutputWr, c)
	// 	if err != nil {
	// 		return err
	// 	}

	for {
		tk, ok := <-s.TickCh
		if !ok {
			break
		}
		newsletters := s.ScheduleStore.Get(tk)

		for _, n := range newsletters {
			l, ok := c.Newsletters[n]
			if !ok {
				return fmt.Errorf("unable to find a configuration for newsletter %v - this is a bug", n)
			}
			err := Run(s.OutputWr, c.Scraping, c.EmailSettings, l)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
