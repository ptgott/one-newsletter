package e2e

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	css "github.com/andybalholm/cascadia"
	"github.com/ptgott/one-newsletter/email"
	"github.com/ptgott/one-newsletter/linksrc"
	"github.com/ptgott/one-newsletter/scrape"
	"github.com/ptgott/one-newsletter/smtptest"
	"github.com/ptgott/one-newsletter/userconfig"
	"github.com/rs/zerolog/log"
)

var (
	appPath string // filled in later--path to the built application
)

// fakeTickChan takes a count of expected email and returns a notification
// schedule and slice of ticks so that each tick satisfies the notification
// schedule.
func fakeTickChan(count int) (userconfig.NotificationSchedule, []time.Time) {
	// Make a notification schedule for a Monday, then create one time.Time for
	// an existing Monday plus each Monday after for a number of weeks equal to
	// count.
	sched := userconfig.NotificationSchedule{
		Weekdays: userconfig.Monday,
		Hour:     0,
	}
	t, err := time.Parse(time.DateOnly, "2025-06-09")
	if err != nil {
		panic(err) // Shouldn't be an error since there's a hardcoded input
	}
	c := make([]time.Time, count)

	for i := 0; i < count; i++ {
		c[i] = t.Add(time.Duration(i*7*24) * time.Hour)
	}
	return sched, c
}

// Check that the number of emails sent is within the expected range.
// Declare a test environment with a number of fake e-publications, run the
// application as a child process, wait for an interval, then stop the
// subprocess to count emails sent.
func TestNewsletterEmailSending(t *testing.T) {
	expectedEmails := 3
	epubs := 3
	linksPerPub := 5
	testenv, err := startTestEnvironment(t, testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
	})
	defer testenv.tearDown()
	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	// One email gets sent right away, so make a tick channel for the rest.
	sched, ticks := fakeTickChan(expectedEmails - 1)

	// Configure link site checks for each fake e-publicaiton we've spun up.
	urls := testenv.urls()
	u := make([]linksrc.Config, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = linksrc.Config{
			URL:  *pu,
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	hostport := strings.Split(testenv.SMTPServer.Address(), ":")
	config, err := createUserConfig(
		userconfig.Meta{
			EmailSettings: email.UserConfig{
				SMTPServerHost: hostport[0],
				SMTPServerPort: hostport[1],
			},
			Newsletters: map[string]userconfig.Newsletter{
				"mynewsletter": userconfig.Newsletter{
					LinkSources: u,
					Schedule:    sched,
				},
			},
			Scraping: userconfig.Scraping{
				StorageDirPath: testenv.tempDirPath,
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	ch := make(chan time.Time, 1)
	store := userconfig.NewScheduleStore()
	store.Add("mynewsletter", sched)
	scrapeConfig := scrape.Config{
		ScheduleStore: store,
		TickCh:        ch,
	}

	go func() {
		for i := range ticks {
			ch <- ticks[i]
		}
		close(ch)
	}()

	scrape.StartLoop(&scrapeConfig, &config)

	ems, err := testenv.SMTPServer.RetrieveEmails(0)

	if err != nil {
		t.Errorf("can't retrieve email from the test SMTP server: %v", err)
	}

	// There should be one email per polling interval, plus the initial
	// email (which is sent right away).
	//
	// Integer division truncates toward zero, so we don't need to find the
	// floor.
	// https://golang.org/ref/spec#Integer_operators
	if len(ems) != expectedEmails {
		t.Errorf(
			"expecting %v emails but got %v",
			expectedEmails,
			len(ems),
		)
	}
}

// Make sure successive emails for the same link site show
// the expected content
func TestNewsletterEmailUpdates(t *testing.T) {
	linksToUpdate := 2
	// Ensure that all emails are the result of polling a single e-publication
	epubs := 1
	linksPerPub := 5
	testenv, err := startTestEnvironment(t, testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
	})
	defer testenv.tearDown()
	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	// Configure link site checks for each fake e-publicaiton we've spun up.
	urls := testenv.urls()
	u := make([]linksrc.Config, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = linksrc.Config{
			URL:  *pu,
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	hostport := strings.Split(testenv.SMTPServer.Address(), ":")
	config, err := createUserConfig(
		userconfig.Meta{
			EmailSettings: email.UserConfig{
				SMTPServerHost: hostport[0],
				SMTPServerPort: hostport[1],
			},
			Newsletters: map[string]userconfig.Newsletter{
				"mynewsletter": userconfig.Newsletter{
					LinkSources: u,
					Schedule: userconfig.NotificationSchedule{
						Weekdays: userconfig.Monday,
						Hour:     12,
					},
				},
			},
			Scraping: userconfig.Scraping{

				StorageDirPath: testenv.tempDirPath,
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Closing the channel immediately since we're relying on the initial
	// email sent in StartLoop.
	ch := make(chan time.Time, 1)
	close(ch)
	scrapeConfig := scrape.Config{
		TickCh: ch,
	}

	scrape.StartLoop(&scrapeConfig, &config)
	// Wait for the application to poll the link site, check for emails,
	// update the application, wait another poll interval, and check
	// for emails again.
	em1, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails before the update: %v", err)
	}
	if len(em1) == 0 {
		t.Fatal("retrieved zero emails before the update")
	}
	before := em1[0] // should just be one email at this point
	log.Info().Msg("updating the mock link sites")
	testenv.update(linksToUpdate)
	ut := time.Now().UnixNano()
	log.Info().Msg("finished updating the mock link sites")

	scrape.StartLoop(&scrapeConfig, &config)
	em2, err := testenv.SMTPServer.RetrieveEmails(ut)
	if err != nil {
		t.Errorf("can't retrieve emails after the update: %v", err)
	}
	if len(em2) == 0 {
		t.Fatal("retrieved zero emails after the update")
	}

	// There should just be one email after filtering by time
	after := em2[0]

	linksBefore := smtptest.ExtractItems(before)
	linksAfter := smtptest.ExtractItems(after)

	if len(linksAfter) != linksToUpdate {
		t.Errorf(
			"expecting %v links in the second email, but got %v",
			linksToUpdate,
			len(linksAfter),
		)
	}

	// Compare link items between successive waves of emails. No items
	// from the first weve should be present in the second.
	// This isn't a very efficienet way to do this, but the number of links
	// in the e2e test will be small.
	for i := range linksAfter {
		for j := range linksBefore {
			if linksAfter[i] == linksBefore[j] {
				t.Errorf(
					"this link is present both before and after a site update: %v",
					linksAfter[i],
				)
			}

		}
	}

}

func TestMaxLinkLimits(t *testing.T) {
	maxLinksInEmail := 5
	epubs := 1
	linksPerPub := 10
	testenv, err := startTestEnvironment(t, testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
	})

	defer testenv.tearDown()

	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	// Configure link site checks for each fake e-publicaiton we've spun up.
	urls := testenv.urls()
	u := make([]linksrc.Config, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = linksrc.Config{
			URL:      *pu,
			Name:     fmt.Sprintf("site-%v", pu.Port()),
			MaxItems: uint(maxLinksInEmail),
		}
	}

	hostport := strings.Split(testenv.SMTPServer.Address(), ":")
	config, err := createUserConfig(
		userconfig.Meta{
			EmailSettings: email.UserConfig{
				SMTPServerHost: hostport[0],
				SMTPServerPort: hostport[1],
			},
			Newsletters: map[string]userconfig.Newsletter{
				"mynewsletter": userconfig.Newsletter{
					LinkSources: u,
					Schedule: userconfig.NotificationSchedule{
						Weekdays: userconfig.Monday,
						Hour:     12,
					},
				},
			},
			Scraping: userconfig.Scraping{

				StorageDirPath: testenv.tempDirPath,
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Closing the channel immediately since we're relying on the initial
	// emails sent in StartLoop.
	ch := make(chan time.Time, 1)
	close(ch)
	scrapeConfig := scrape.Config{
		TickCh: ch,
	}

	scrape.StartLoop(&scrapeConfig, &config)
	em, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	if len(em) == 0 {
		t.Fatal("retrieved zero emails")
	}
	bod := em[0] // should just be one email at this point

	links := smtptest.ExtractItems(bod)

	if len(links) > maxLinksInEmail {
		t.Errorf(
			"expecting %v links in the email, but got %v",
			maxLinksInEmail,
			len(links),
		)
	}

}

// totalBadgerDataFileSize gets the total size of all the VLOG and SSt files in
// a directory in bytes.
//
// We need this because FileInfo.Size() acts in a system dependent way for
// directories.
// See: https://golang.org/pkg/io/fs/#FileInfo
func totalBadgerDataFileSize(dirPath string) float64 {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}
	var size int64
	for i := range entries {
		if !entries[i].IsDir() &&
			(strings.HasSuffix(entries[i].Name(), "vlog") || strings.HasSuffix(entries[i].Name(), "sst")) {
			fi, err := entries[i].Info()
			if err != nil {
				panic(err)
			}
			size += fi.Size()
		}
	}
	return float64(size)
}

// Make sure that an email is still sent if the only scrape config contains
// invalid CSS. This test exists because one site with a config that included
// an ambiguous selector seems to have caused the application to deadlock.
func TestEmailSendingWithBadScrapeConfig(t *testing.T) {
	expectedEmails := 2
	epubs := 1
	linksPerPub := 10
	testenv, err := startTestEnvironment(t, testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
	})

	defer testenv.tearDown()

	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	// One email gets sent right away, so make a tick channel for the rest.
	sched, ticks := fakeTickChan(expectedEmails - 1)

	// Configure link site checks for each fake e-publicaiton we've spun up.
	urls := testenv.urls()
	u := make([]linksrc.Config, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = linksrc.Config{
			URL:      *pu,
			Name:     fmt.Sprintf("site-%v", pu.Port()),
			MaxItems: 5,
			// "ul" is ambiguous, since each link item has the selector
			// "ul li"
			ItemSelector:    css.MustCompile("ul"),
			CaptionSelector: css.MustCompile("p"),
			LinkSelector:    css.MustCompile("a"),
		}
	}

	hostport := strings.Split(testenv.SMTPServer.Address(), ":")
	config, err := createUserConfig(
		userconfig.Meta{
			EmailSettings: email.UserConfig{
				SMTPServerHost: hostport[0],
				SMTPServerPort: hostport[1],
			},
			Newsletters: map[string]userconfig.Newsletter{
				"mynewsletter": userconfig.Newsletter{
					LinkSources: u,
					Schedule: userconfig.NotificationSchedule{
						Weekdays: userconfig.Monday,
						Hour:     12,
					},
				},
			},
			Scraping: userconfig.Scraping{
				StorageDirPath: testenv.tempDirPath,
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	ch := make(chan time.Time, 1)
	store := userconfig.NewScheduleStore()
	store.Add("mynewsletter", sched)
	scrapeConfig := scrape.Config{
		ScheduleStore: store,
		TickCh:        ch,
	}

	go func() {
		for i := range ticks {
			ch <- ticks[i]
		}
		close(ch)
	}()

	scrape.StartLoop(&scrapeConfig, &config)

	em, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	// Expecting an iteration limit of one, plus the email that gets sent
	// right away.
	if len(em) != 2 {
		t.Fatalf("expected to receive one email, but got %v", len(em))
	}
}

// Test that the -test flag causes email bodies to be printed to stdout,
// and that no emails are sent.
func TestTestModeFlag(t *testing.T) {
	// Ensure that all emails are the result of polling a single e-publication
	epubs := 1
	linksPerPub := 5
	testenv, err := startTestEnvironment(t, testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
	})

	defer testenv.tearDown()

	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	// Configure link site checks for each fake e-publicaiton we've spun up.
	urls := testenv.urls()
	u := make([]linksrc.Config, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = linksrc.Config{
			URL:  *pu,
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	hostport := strings.Split(testenv.SMTPServer.Address(), ":")
	config, err := createUserConfig(
		userconfig.Meta{
			EmailSettings: email.UserConfig{
				SMTPServerHost: hostport[0],
				SMTPServerPort: hostport[1],
			},
			Newsletters: map[string]userconfig.Newsletter{
				"mynewsletter": userconfig.Newsletter{
					LinkSources: u,
					Schedule: userconfig.NotificationSchedule{
						Weekdays: userconfig.Monday,
						Hour:     12,
					},
				},
			},
			Scraping: userconfig.Scraping{
				TestMode:       true, // This is important here
				StorageDirPath: testenv.tempDirPath,
				NewsletterName: "mynewsletter",
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	var msg bytes.Buffer

	// Closing the channel immediately since we're relying on the initial
	// email sent in StartLoop.
	ch := make(chan time.Time, 1)
	close(ch)
	scrapeConfig := scrape.Config{
		TickCh:   ch,
		OutputWr: &msg,
	}

	scrape.StartLoop(&scrapeConfig, &config)

	em1, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	if len(em1) != 0 {
		t.Fatalf("expected to receive zero emails but got %v", em1)
	}

	o, err := io.ReadAll(&msg)
	if err != nil {
		t.Fatalf("could not read from the scraper output: %v", err)
	}

	links := smtptest.ExtractItems(string(o))
	if len(links) != epubs*linksPerPub {
		t.Errorf(
			"expecting %v links via stdout, but got %v",
			epubs*linksPerPub,
			len(links),
		)
	}
}

func TestOneOffFlag(t *testing.T) {
	epubs := 3
	linksPerPub := 5
	testenv, err := startTestEnvironment(t, testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
	})

	defer testenv.tearDown()

	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	// Configure link site checks for each fake e-publicaiton we've spun up.
	urls := testenv.urls()
	u := make([]linksrc.Config, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = linksrc.Config{
			URL:  *pu,
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	hostport := strings.Split(testenv.SMTPServer.Address(), ":")
	config, err := createUserConfig(
		userconfig.Meta{
			EmailSettings: email.UserConfig{
				SMTPServerHost: hostport[0],
				SMTPServerPort: hostport[1],
			},
			Newsletters: map[string]userconfig.Newsletter{
				"mynewsletter": userconfig.Newsletter{
					LinkSources: u,
					Schedule: userconfig.NotificationSchedule{
						Weekdays: userconfig.Monday,
						Hour:     12,
					},
				},
			},
			Scraping: userconfig.Scraping{
				StorageDirPath: testenv.tempDirPath,
				OneOff:         true, // This is important here
				NewsletterName: "mynewsletter",
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Closing the channel immediately since we're relying on the initial
	// email sent in StartLoop.
	ch := make(chan time.Time, 1)
	close(ch)
	scrapeConfig := scrape.Config{
		TickCh: ch,
	}

	dbBefore := totalBadgerDataFileSize(testenv.tempDirPath)

	// The -oneoff flag should cause the scraper loop to run as a one-off
	// job
	scrape.StartLoop(&scrapeConfig, &config)

	dbAfter := totalBadgerDataFileSize(testenv.tempDirPath)

	if dbAfter > dbBefore {
		t.Errorf(
			"the one-off command must not write to the database: expecting data directory size to be %v but got %v",
			dbBefore,
			dbAfter,
		)
	}

	ems, err := testenv.SMTPServer.RetrieveEmails(0)

	if err != nil {
		t.Errorf("can't retrieve email from the test SMTP server: %v", err)
	}

	// The -oneoff flag should cause only one email to be sent
	if len(ems) != 1 {
		t.Errorf(
			"expecting one but got %v",
			len(ems),
		)
	}

}

func TestOneOffFlagWithNoEmailFlag(t *testing.T) {
	epubs := 3
	linksPerPub := 5
	testenv, err := startTestEnvironment(t, testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
	})

	defer testenv.tearDown()

	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	// Configure link site checks for each fake e-publicaiton we've spun up.
	urls := testenv.urls()
	u := make([]linksrc.Config, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = linksrc.Config{
			URL:  *pu,
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	hostport := strings.Split(testenv.SMTPServer.Address(), ":")
	config, err := createUserConfig(
		userconfig.Meta{
			EmailSettings: email.UserConfig{
				SMTPServerHost: hostport[0],
				SMTPServerPort: hostport[1],
			},
			Newsletters: map[string]userconfig.Newsletter{
				"mynewsletter": userconfig.Newsletter{
					LinkSources: u,
					Schedule: userconfig.NotificationSchedule{
						Weekdays: userconfig.Monday,
						Hour:     12,
					},
				},
			},
			Scraping: userconfig.Scraping{
				// Note that both TestMode and OneOff are true here.
				TestMode:       true,
				OneOff:         true,
				NewsletterName: "mynewsletter",
				StorageDirPath: testenv.tempDirPath,
			},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	var msg bytes.Buffer
	// Closing the channel immediately since we're relying on the initial
	// email sent in StartLoop.
	ch := make(chan time.Time, 1)
	close(ch)
	scrapeConfig := scrape.Config{
		TickCh:   ch,
		OutputWr: &msg,
	}

	// The -oneoff flag should cause the scraper loop to run as a one-off
	// job
	scrape.StartLoop(&scrapeConfig, &config)

	ems, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	if len(ems) != 0 {
		t.Fatalf("expected to receive zero emails but got %v", ems)
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(&msg)
	if err != nil {
		t.Fatalf("could not read from the command output: %v", err)
	}

	links := smtptest.ExtractItems(buf.String())
	if len(links) != epubs*linksPerPub {
		t.Errorf(
			"expecting %v links via stdout, but got %v",
			epubs*linksPerPub,
			len(links),
		)
	}

}
