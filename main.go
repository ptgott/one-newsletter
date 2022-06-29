package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/ptgott/one-newsletter/html"
	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/ptgott/one-newsletter/storage"
	"github.com/ptgott/one-newsletter/userconfig"

	"github.com/rs/zerolog/log"
)

// runScrape conducts a single scrape and email cycle, sending any errors to
// the error channel ec. It reads the user config anew at the beginning of
// each cycle.
func runScrape(ec chan error, config userconfig.Meta) {

	httpClient := http.Client{
		// Determined arbitrarily. We don't want to wait forever for a
		// request to complete, but the cadence of the newsletter means
		// that a minute of extra waiting is probably okay.
		Timeout: time.Duration(60) * time.Second,
	}

	// Create a new db instance per scrape so we can close
	// the db after the scrapers are finished and ensure
	// disk writes.
	db, err := storage.SetUpDB(
		config.Scraping.StorageDirPath,
		config.Scraping.Interval,
	)

	if err != nil {
		log.Error().Err(err).Msg("unable to set up the database")
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
	err = db.Cleanup()
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

	if *noEmail {
		os.Stdout.Write([]byte(bod))
	} else {
		err = config.EmailSettings.SendNewsletter([]byte(txt), []byte(bod))
		if err != nil {
			log.Error().Err(err).Msg("error sending an email")
		}
	}

	// We're only doing this once, so get out of the main loop
	if *oneOff {
		close(errCh)
		return
	}

}

func main() {
	// Log with filename and line number. This writes to stderr, so it should
	// be thread safe.
	// https://github.com/rs/zerolog/blob/7ccd4c940bf8a02fcc5f10e5475f9d3daff04d57/log/log.go#L13
	log.Logger = log.With().Caller().Logger()

	// Intercept interrupts so we can get more visibility into them.
	// One goroutine listens exclusively for interrupts so we can
	// handle them before the main application loop in case of
	// setup issues.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func(c chan os.Signal) {
		<-sigCh
		log.Info().Msg("interrupt: exiting")
		os.Exit(0)
	}(sigCh)

	configPath := flag.String(
		"config",
		"./config.yaml",
		"path to a JSON or YAML file containing your configuration",
	)
	noEmail := flag.Bool(
		"noemail",
		false,
		"print email body HTML to stdout instead of sending it",
	)
	oneOff := flag.Bool(
		"oneoff",
		false,
		"run the scrapers once and (unless -noemail is present) send one email",
	)
	flag.Parse()

	log.Info().
		Str("configPath", *configPath).
		Msg("starting the application")

	f, err := os.Open(*configPath)

	if err != nil {
		log.Error().
			Str("config-path", *configPath).
			Err(err).
			Msg("We can't open the application config file")
		os.Exit(1)
	}

	config, err := userconfig.Parse(f)

	if err != nil {
		log.Error().
			Err(err).
			Msg("Problem parsing your config")
		os.Exit(1)
	}

	log.Info().Str("configPath", *configPath).Msg("successfully validated the config")

	// Since this is a one-off or a test, set the data directory to an
	// empty string to disable database operations.
	if *oneOff || *noEmail {
		config.Scraping.StorageDirPath = ""
		log.Debug().Msg(
			"disabling database operations",
		)
	}

	// Declare channels between the main goroutine and the scrapers
	errCh := make(chan error) // errors to print
	scrapeCadence := time.NewTicker(config.Scraping.Interval)

	go func(tc <-chan time.Time, ec chan error) {
		// Run the first scrape immediately
		runScrape(ec)

		// enter the main scraping/email sending loop
		for !*oneOff {

			<-tc

			go runScrape(ec)
		}
	}(scrapeCadence, errCh)

	// At this point, the main goroutine blocks until there's an error
	for {
		err, ok := <-errCh
		// There's no need for the error channel anymore, so we stop
		// looping and let the rest of the program complete.
		if !ok {
			break
		} else {
			log.Error().Err(err).Msg("error gathering links to email")
		}
	}
}
