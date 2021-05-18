package main

import (
	"divnews/email"
	"divnews/html"
	"divnews/linksrc"
	"divnews/poller"
	"divnews/storage"
	"divnews/userconfig"
	"flag"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

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
		"Path to a JSON or YAML file containing your configuration",
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

	smtpCl, err := email.NewSMTPClient(config.EmailSettings)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Problem setting up the email client")
		os.Exit(1)
	}
	log.Info().Msg("set up the SMTP client successfully")

	errCh := make(chan error) // errors to print

	var httpClient poller.Client
	scrapeCadence := time.NewTicker(config.PollSettings.Interval)

	for {
		select {
		case <-scrapeCadence.C:
			db, err := storage.NewBadgerDB(config.StorageSettings)
			if err != nil {
				log.Error().
					Err(err).
					Msg("problem connecting to the database")
				continue // Maybe this was a transient error? Log it and move on.
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
					ec chan error,
				) {
					defer g.Done()
					// Try the scrape request only once. If we get a non-2xx
					// response, it's probably not something we can expect to
					// clear up after retrying.
					r, err := httpClient.Client.Get(lc.URL.String())
					if err != nil {
						ec <- err
						return
					}
					defer r.Body.Close()
					s, err := linksrc.NewSet(r.Body, lc, r.StatusCode)
					if err != nil {
						ec <- err
						return
					}

					bc <- s

				}(ls, &wg, emailBuildCh, errCh)
			}
			wg.Wait()
			close(emailBuildCh)
			log.Info().
				Msg("done with one round of scraping")
			for set := range emailBuildCh {
				newSet := linksrc.Set{
					Name:   set.Name,
					Items:  []linksrc.LinkItem{},
					Status: set.Status,
				}
				// See if any items are missing in the db. If so, store them
				// and add them to a new email body.
				for _, item := range set.Items {
					// Read returns a "key not found" error if a key is not found.
					// https://pkg.go.dev/github.com/dgraph-io/badger#Txn.Get
					_, err := db.Read(item.Key())
					if err != nil {
						newSet.Items = append(newSet.Items, item)
					}
					log.Info().Msg("storing a link item in the database")
					err = db.Put(item.NewKVEntry())
					if err != nil {
						log.Error().
							Err(err).
							Msg("error saving a link item")
						continue
					}
				}
				d.Add(newSet)
				log.Info().
					Int("itemCount", len(newSet.Items)).
					Str("setName", newSet.Name).
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
			bod, err := d.GenerateBody()
			if err != nil {
				log.Error().Err(err).Msg("error generating an email body")
				continue
			}
			txt, err := d.GenerateText()
			if err != nil {
				log.Error().Err(err).Msg("error generating an email plaintext")
				continue
			}
			log.Info().
				Msg("attempting to send an email")
			err = smtpCl.SendNewsletter([]byte(txt), []byte(bod))
			if err != nil {
				log.Error().Err(err).Msg("error sending an email")
				continue
			}
		case err := <-errCh:
			log.Error().Err(err).Msg("error gathering links to email")
		}
	}
}
