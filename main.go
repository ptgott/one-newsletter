package main

import (
	"flag"
	"os"
	"os/signal"
	"time"

	"github.com/ptgott/one-newsletter/scrape"
	"github.com/ptgott/one-newsletter/userconfig"

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

	config.Scraping.OneOff = *oneOff
	config.Scraping.NoEmail = *noEmail
	// Since this is a one-off or a test, set the data directory to an
	// empty string to disable database operations.
	if *oneOff || *noEmail {
		config.Scraping.StorageDirPath = ""
		log.Debug().Msg(
			"disabling database operations",
		)
	}

	scrapeCadence := time.NewTicker(config.Scraping.Interval)

	scrapeConfig := scrape.Config{
		TickCh:   scrapeCadence.C,
		ErrCh:    make(chan error), // errors to print
		OutputWr: os.Stdout,        // write to stdout if the -no-email flag is given
		StopCh:   nil,              // since we simply exit on a SIGINT
	}

	go scrape.StartLoop(&scrapeConfig, config)

	// At this point, the main goroutine blocks until there's an error
	for {
		err := <-scrapeConfig.ErrCh
		log.Error().Err(err).Msg("error gathering links to email")
	}

}
