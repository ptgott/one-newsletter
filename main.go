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
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func main() {
	// Log with filename and line number
	log.Logger = log.With().Caller().Logger()

	configPath := flag.String(
		"config",
		"./config.yaml",
		"Path to a JSON or YAML file containing your configuration",
	)
	flag.Parse()

	log.Info().
		Str("config-path", *configPath).
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

	c, err := email.NewSMTPClient(config.EmailSettings)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Problem setting up the email client")
		os.Exit(1)
	}

	var emailCh chan string // email bodies to send
	var errCh chan error
	var storeCh chan storage.KVEntry
	var httpClient poller.Client
	scrapeCadence := time.NewTicker(config.PollSettings.Interval)
	cleanupCadence := time.NewTicker(config.StorageSettings.CleanupInterval)

	db, err := storage.NewBadgerDB(config.StorageSettings)
	if err != nil {
		log.Error().
			Err(err).
			Msg("problem connecting to the database")
		os.Exit(1)
	}
	defer db.Close()

	select {
	case bod := <-emailCh:
		err = c.Send(bod)
		if err != nil {
			errCh <- err
			return
		}
	case <-scrapeCadence.C:
		var wg sync.WaitGroup
		var d html.EmailData
		wg.Add(len(config.LinkSources))
		for _, ls := range config.LinkSources {
			// TODO: Consider extracting this
			go func(lc linksrc.Config, wg *sync.WaitGroup) {
				defer wg.Done()
				r, err := httpClient.Poll(lc.URL)
				if err != nil {
					errCh <- err
					return
				}
				s, err := linksrc.NewSet(r, lc)
				if err != nil {
					errCh <- err
					return
				}

				latest, err := s.NewKVEntry()
				if err != nil {
					errCh <- err
					return
				}

				// Send latest to storeCh before diffing with an earlier
				// iteration. This shouldn't matter for subsequent iterations.
				storeCh <- latest

				earlier, err := db.Read(latest.Key)

				// Skip the comparison if there's no earlier Set for this key
				if err == nil {
					// TODO: The if statements in this block need to break somehow!
					olds, err := linksrc.Deserialize(earlier)
					// Log the error for observation but move on, since we can
					// ignore the earlier Set
					if err != nil {
						errCh <- err
						goto add
					}
					news, err := s.NewSince(olds)
					if err != nil {
						errCh <- err
						goto add
					}
					s = news
				}
			add:
				d.Add(s)
			}(ls, &wg)
		}
		wg.Wait()
		bod, err := d.GenerateBody()
		if err != nil {
			errCh <- err
			return
		}
		emailCh <- bod
	case <-cleanupCadence.C:
		err := db.Cleanup()
		log.Error().Err(err).Msg("error cleaning up the database")
	case kve := <-storeCh:
		err = db.Put(kve)
		if err != nil {
			errCh <- flag.ErrHelp
			return
		}
	case err := <-errCh:
		log.Error().Err(err).Msg("error sending email")
	}
}
