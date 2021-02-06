package main

import (
	"divnews/email"
	"divnews/html"
	"divnews/linksrc"
	"divnews/poller"
	"divnews/storage"
	"divnews/userconfig"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"
)

func main() {
	configPath := flag.String(
		"config",
		"./config.yaml",
		"Path to a JSON or YAML file containing your configuration",
	)
	flag.Parse()

	f, err := os.Open(*configPath)

	if err != nil {
		fmt.Printf(
			"We can't open the config file at %v: %v\n",
			configPath,
			err,
		)
		os.Exit(1)
	}

	config, err := userconfig.Parse(f)

	if err != nil {
		fmt.Printf("Problem parsing your config: %v\n", err)
		os.Exit(1)
	}

	c, err := email.NewSMTPClient(config.EmailSettings)
	if err != nil {
		fmt.Printf("Problem setting up the email client: %v", err)
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
		fmt.Printf("Problem connecting to the database: %v", err)
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
		db.Cleanup()
		// TODO: Log any errors. For now, we can ignore them as long
		// as the number of link sources is small while observing
		// possible failure modes
	case kve := <-storeCh:
		err = db.Put(kve)
		if err != nil {
			errCh <- flag.ErrHelp
			return
		}
	case err := <-errCh:
		fmt.Println(err)
	}
}
