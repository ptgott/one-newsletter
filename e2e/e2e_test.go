package e2e

import (
	"bytes"
	"divnews/smtptest"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"gonum.org/v1/gonum/stat"
)

var (
	appPath string // filled in later--path to the built application
)

func TestMain(m *testing.M) {
	// We need to build the application before we can run it. While
	// executing "go run" in the test environment seems like a nice
	// cross-platform choice, the main "go run" process isn't actually what
	// executes the program. This means that when the test environment
	// terminates the "go run" process, it leaves an orphan process that
	// can't be managed by the test environment.
	rand.Seed(time.Now().UnixNano())
	appPath = fmt.Sprintf("./app%v", rand.Intn(1000))
	bld := exec.Command("go", "build", "-o", appPath, "../main.go")
	err := bld.Run()
	if err != nil {
		panic(fmt.Sprintf("can't build the application: %v", err))
	}

	err = os.Chmod(appPath, 0777)
	if err != nil {
		panic(fmt.Sprintf("can't change the application permissions"))
	}

	s := m.Run()
	os.Remove(appPath)
	os.Exit(s)
}

// Check that the number of emails sent is within the expected range.
// Declare a test environment with a number of fake e-publications, run the
// application as a child process, wait for an interval, then stop the
// subprocess to count emails sent.
func TestNewsletterEmailSending(t *testing.T) {
	stopIntervalS := 11
	pollIntervalS := 5
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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:  urls[i],
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Start(); err != nil {
		t.Fatalf("couldn't start the app: %v", err)
	}

	time.Sleep(time.Duration(stopIntervalS) * time.Second)

	err = cmd.Process.Signal(os.Interrupt)

	// At this point you need to find the process and kill it manually.
	// This messes up the test, so we panic.
	if err != nil {
		t.Fatalf("pid %v could not be interrupted", cmd.Process.Pid)
	}

	// it's okay for the application to exit with an error--we want to proceed
	// with the test suite so we can get visibility into those errors
	if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("couldn't stop the application process: %v", err)
	}

	ems, err := testenv.SMTPServer.RetrieveEmails(0)

	if err != nil {
		t.Errorf("can't retrieve email from the test SMTP server: %v", err)
	}

	// There should be one email per polling interval.
	//
	// Integer division truncates toward zero, so we don't need to
	// find the floor.
	// https://golang.org/ref/spec#Integer_operators
	expectedLen := stopIntervalS / pollIntervalS
	if len(ems) != int(expectedLen) {
		t.Errorf(
			"expecting %v emails but got %v",
			expectedLen,
			len(ems),
		)
	}

}

// Make sure successive emails for the same link site show
// the expected content
func TestNewsletterEmailUpdates(t *testing.T) {
	// i.e., poll the site once, update the site, poll it again,
	// and stop.
	updateIntervalS := 6
	stopIntervalS := 11
	pollIntervalS := 5
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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:  urls[i],
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Start(); err != nil {
		t.Fatalf("couldn't start the app: %v", err)
	}

	// Wait for the application to poll the link site, check for emails,
	// update the application, wait another poll interval, and check
	// for emails again.
	time.Sleep(time.Duration(updateIntervalS) * time.Second)
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
	time.Sleep(time.Duration(stopIntervalS-updateIntervalS) * time.Second)
	err = cmd.Process.Signal(os.Interrupt)

	// At this point you need to find the process and kill it manually.
	// This messes up the test, so we panic.
	if err != nil {
		t.Fatalf("pid %v could not be interrupted", cmd.Process.Pid)
	}

	// it's okay for the application to exit with an error--we want to proceed
	// with the test suite so we can get visibility into those errors
	if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("couldn't stop the application process: %v", err)
	}

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
	stopIntervalS := 7
	pollIntervalS := 5
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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:      urls[i],
			Name:     fmt.Sprintf("site-%v", pu.Port()),
			MaxItems: 5,
		}
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Start(); err != nil {
		t.Fatalf("couldn't start the app: %v", err)
	}

	// Wait for the application to poll the link site, check for emails,
	// update the application, wait another poll interval, and check
	// for emails again.
	time.Sleep(time.Duration(stopIntervalS) * time.Second)
	err = cmd.Process.Signal(os.Interrupt)
	// At this point you need to find the process and kill it manually.
	// This messes up the test, so we panic.
	if err != nil {
		t.Fatalf("pid %v could not be interrupted", cmd.Process.Pid)
	}

	// it's okay for the application to exit with an error--we want to proceed
	// with the test suite so we can get visibility into those errors
	if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("couldn't stop the application process: %v", err)
	}

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

func TestDBcleanup(t *testing.T) {
	pollIntervalS := 5
	pollCycles := 10
	// just a bit more than the pollInterval
	diskCheckIntervalMS := 5100

	epubs := 10
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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:  urls[i],
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Start(); err != nil {
		t.Fatalf("couldn't start the app: %v", err)
	}

	fileSizes := make([]float64, pollCycles, pollCycles)
	for i := range fileSizes {
		time.Sleep(time.Duration(diskCheckIntervalMS) * time.Millisecond)
		fileSizes[i] = totalBadgerDataFileSize(testenv.tempDirPath)
		testenv.update(linksPerPub)
	}

	err = cmd.Process.Signal(os.Interrupt)

	// At this point you need to find the process and kill it manually.
	// This messes up the test, so we panic.
	if err != nil {
		t.Fatalf("pid %v could not be interrupted", cmd.Process.Pid)
	}

	// it's okay for the application to exit with an error--we want to proceed
	// with the test suite so we can get visibility into those errors
	if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("couldn't stop the application process: %v", err)
	}

	// The test assertion is based on the standard deviation of the file sizes,
	// since this is in the same unit as the file size (bytes).
	// We ignore the first two values because they aren't expected to reflect
	// the baseline state  of the application's disk usage.
	// Garbage collection should stabilize the data directory file size over time.
	stddev := stat.StdDev(fileSizes[2:], nil)

	// We want the data directory to vary by, at most, 50 percent of the
	// initial (post-setup) value in bytes. The initial value won't be that high,
	// so 50 percent is a pretty good arbitrary limit over time.
	var maxStdDev float64 = fileSizes[2] * .50

	if stddev > maxStdDev {
		t.Errorf(
			"expected data directory size to vary by less than %v bytes, but got %v, with post-setup file sizes: %v",
			maxStdDev,
			stddev,
			fileSizes[2:],
		)
	}

}

// Make sure that an email is still sent if the only scrape config contains
// invalid CSS. This test exists because one site with a config that included
// an ambiguous selector seems to have caused the application to deadlock.
func TestEmailSendingWithBadScrapeConfig(t *testing.T) {
	stopIntervalS := 8
	pollIntervalS := 5
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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:      urls[i],
			Name:     fmt.Sprintf("site-%v", pu.Port()),
			MaxItems: 5,
			// "ul" is ambiguous, since each link items has the selector
			// "ul li"
			SelectorsOverride: `      itemSelector: ul
      captionSelector: p
      linkSelector: a`,
		}
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err = cmd.Start(); err != nil {
		t.Fatalf("couldn't start the app: %v", err)
	}

	// Wait for the application to poll the link site and check for emails
	time.Sleep(time.Duration(stopIntervalS) * time.Second)
	err = cmd.Process.Signal(os.Interrupt)
	// At this point you need to find the process and kill it manually.
	// This messes up the test, so we panic.
	if err != nil {
		t.Fatalf("pid %v could not be interrupted", cmd.Process.Pid)
	}

	// it's okay for the application to exit with an error--we want to proceed
	// with the test suite so we can get visibility into those errors
	if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("couldn't stop the application process: %v", err)
	}

	em, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	if len(em) != 1 {
		t.Fatalf("expected to receive one email, but got %v", len(em))
	}

}

// Test that the -noemail flag causes email bodies to be printed to stdout,
// and that no emails are sent.
func TestNoEmailFlag(t *testing.T) {
	stopIntervalS := 6
	pollIntervalS := 5

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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:  urls[i],
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	// We'll still fire up an SMTP server, but we shouldn't be sending anything
	// to it.
	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf(
			"-config=%v/%v",
			testenv.tempDirPath,
			"config.yaml",
		),
		"-noemail",
	)

	cmd.Stderr = os.Stderr
	// Capture stdout in a buffer so we can capture the results
	var cmdOut bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &cmdOut)

	if err = cmd.Start(); err != nil {
		t.Fatalf("couldn't start the app: %v", err)
	}

	time.Sleep(time.Duration(stopIntervalS) * time.Second)
	err = cmd.Process.Signal(os.Interrupt)

	// At this point you need to find the process and kill it manually.
	// This messes up the test, so we panic.
	if err != nil {
		t.Fatalf("pid %v could not be interrupted", cmd.Process.Pid)
	}

	// it's okay for the application to exit with an error--we want to proceed
	// with the test suite so we can get visibility into those errors
	if err := cmd.Wait(); err != nil && !strings.Contains(err.Error(), "exit status") {
		t.Fatalf("couldn't stop the application process: %v", err)
	}

	em1, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	if len(em1) != 0 {
		t.Fatalf("expected to receive zero emails but got %v", em1)
	}

	o, err := io.ReadAll(&cmdOut)
	if err != nil {
		t.Fatalf("could not read from the command output: %v", err)
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
	pollIntervalS := 5
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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:  urls[i],
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
		"-oneoff",
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	dbBefore := totalBadgerDataFileSize(testenv.tempDirPath)

	// The -oneoff flag should cause the command to become a discrete job
	if err = cmd.Run(); err != nil {
		t.Fatalf("couldn't run the app: %v", err)
	}

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

	// The -oneoff flag hsould cause only one email to be sent
	if len(ems) != 1 {
		t.Errorf(
			"expecting one but got %v",
			len(ems),
		)
	}

}

func TestOneOffFlagWithNoEmailFlag(t *testing.T) {
	pollIntervalS := 5
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
	u := make([]mockLinksrcInfo, len(urls), len(urls))
	for i := range urls {
		// not expecting errors since these URLs are guaranteed to be
		// for running servers, and don't come from user input
		pu, _ := url.Parse(urls[i])

		u[i] = mockLinksrcInfo{
			URL:  urls[i],
			Name: fmt.Sprintf("site-%v", pu.Port()),
		}
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			SMTPServerAddress: testenv.SMTPServer.Address(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
		"-oneoff",
		"-noemail",
	)

	cmd.Stderr = os.Stderr
	// Capture stdout in a buffer so we can capture the results
	var cmdOut bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &cmdOut)

	// The -oneoff flag should cause the command to become a discrete job
	if err = cmd.Run(); err != nil {
		t.Fatalf("couldn't run the app: %v", err)
	}

	ems, err := testenv.SMTPServer.RetrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	if len(ems) != 0 {
		t.Fatalf("expected to receive zero emails but got %v", ems)
	}

	o, err := io.ReadAll(&cmdOut)
	if err != nil {
		t.Fatalf("could not read from the command output: %v", err)
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
