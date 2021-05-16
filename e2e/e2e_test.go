package e2e

import (
	"encoding/json"
	"fmt"
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
	mailHogPath     string // path to the MailHog executable taken from user config
	mailHogSMTPPort int
	mailHogHTTPPort int
	appPath         string // filled in later--path to the built application
)

func TestMain(m *testing.M) {
	j, err := os.ReadFile("../e2e_config.json")
	if err != nil {
		panic(fmt.Sprintf("can't open the e2e config file: %v", err))
	}
	var opts map[string]interface{}
	err = json.Unmarshal(j, &opts)
	if err != nil {
		panic(fmt.Sprintf("can't parse the e2e test config as json: %v", err))
	}
	v, ok := opts["mailhog_path"]
	if !ok {
		panic("the e2e config file must specify a mailhog_path")
	}
	if mailHogPath, ok = v.(string); !ok {
		panic("mailhog_path must be a string")
	}
	n, ok := opts["mailhog_smtp_port"]
	if !ok {
		panic("the e2e config file must specify a mailhog_smtp_port")
	}
	n2, ok := n.(float64)
	if !ok {
		panic("mailhog_smtp_port must be a number")
	}
	mailHogSMTPPort = int(n2)
	k, ok := opts["mailhog_http_port"]
	if !ok {
		panic("the e2e config file must specify a mailhog_http_port")
	}
	k2, ok := k.(float64)
	if !ok {
		panic("mailhog_http_port must be a number")
	}
	mailHogHTTPPort = int(k2)

	// We need to build the application before we can run it. While
	// executing "go run" in the test environment seems like a nice
	// cross-platform choice, the main "go run" process isn't actually what
	// executes the program. This means that when the test environment
	// terminates the "go run" process, it leaves an orphan process that
	// can't be managed by the test environment.
	rand.Seed(time.Now().UnixNano())
	appPath = fmt.Sprintf("./app%v", rand.Intn(1000))
	bld := exec.Command("go", "build", "-o", appPath, "../main.go")
	err = bld.Run()
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
	testenv, err := startTestEnvironment(testEnvironmentConfig{
		numHTTPServers:  epubs,
		numLinks:        linksPerPub,
		mailHogPath:     mailHogPath,
		mailHogHTTPPort: mailHogHTTPPort,
		mailHogSMTPPort: mailHogSMTPPort,
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
			SMTPServerAddress: testenv.testSMTPServer.smtpAddress(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
			KeyTTL:            "168h", // no cleanup expected during the test
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

	ems, err := testenv.retrieveEmails(0)

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
	testenv, err := startTestEnvironment(testEnvironmentConfig{
		numHTTPServers:  epubs,
		numLinks:        linksPerPub,
		mailHogPath:     mailHogPath,
		mailHogHTTPPort: mailHogHTTPPort,
		mailHogSMTPPort: mailHogSMTPPort,
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
			SMTPServerAddress: testenv.testSMTPServer.smtpAddress(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
			KeyTTL:            "168h", // no cleanup expected during the test
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
	em1, err := testenv.retrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails before the update: %v", err)
	}
	if len(em1) == 0 {
		t.Fatal("retrieved zero emails before the update")
	}
	before := em1[0] // should just be one email at this point

	log.Info().Msg("updating the mock link sites")
	testenv.update(linksToUpdate)
	ut := time.Now().Unix()
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

	em2, err := testenv.retrieveEmails(ut)
	if err != nil {
		t.Errorf("can't retrieve emails after the update: %v", err)
	}
	if len(em2) == 0 {
		t.Fatal("retrieved zero emails after the update")
	}

	// There should just be one email after filtering by time
	after := em2[0]

	linksBefore := extractItems(before)
	linksAfter := extractItems(after)

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
	testenv, err := startTestEnvironment(testEnvironmentConfig{
		numHTTPServers:  epubs,
		numLinks:        linksPerPub,
		mailHogPath:     mailHogPath,
		mailHogHTTPPort: mailHogHTTPPort,
		mailHogSMTPPort: mailHogSMTPPort,
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
			SMTPServerAddress: testenv.testSMTPServer.smtpAddress(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
			KeyTTL:            "168h", // no cleanup expected during the test
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

	em, err := testenv.retrieveEmails(0)
	if err != nil {
		t.Errorf("could not retrieve emails: %v", err)
	}
	if len(em) == 0 {
		t.Fatal("retrieved zero emails")
	}
	bod := em[0] // should just be one email at this point

	links := extractItems(bod)

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
			fmt.Printf("file to stat: %v with size: %v\n", entries[i].Name(), fi.Size())
			size += fi.Size()
		}
	}
	return float64(size)
}

func TestDBcleanup(t *testing.T) {
	pollIntervalS := 5
	waitPaddingMS := 300 // to make sure we're done with scraping before we stat a directory
	// This is short so we can guarantee cleanup
	keyTTLms := 100
	pollCycles := 6

	epubs := 10
	linksPerPub := 10

	testenv, err := startTestEnvironment(testEnvironmentConfig{
		numHTTPServers:  epubs,
		numLinks:        linksPerPub,
		mailHogPath:     mailHogPath,
		mailHogHTTPPort: mailHogHTTPPort,
		mailHogSMTPPort: mailHogSMTPPort,
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
			SMTPServerAddress: testenv.testSMTPServer.smtpAddress(),
			LinkSources:       u,
			StorageDir:        testenv.tempDirPath,
			PollInterval:      fmt.Sprintf("%vs", pollIntervalS),
			KeyTTL:            fmt.Sprintf("%vms", keyTTLms),
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
		// Wait for one polling interval, then update all e-publications. This means that the next polling
		// interval should trigger a wave of new database writes.
		time.Sleep(time.Duration(pollIntervalS)*time.Second + time.Duration(waitPaddingMS)*time.Millisecond)
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

	// The test assertion is based on the variance of the file sizes. We ignore
	// the first value because it differs unreliably between test runs, e.g.,
	// due to test setup. Garbage collection should stabilize the data directory
	// file size over time.
	v := stat.Variance(fileSizes[1:], nil)
	var maxVariance float64 = 100 // bytes

	if v > maxVariance {
		t.Errorf(
			"expected data directory size to vary by less than %v bytes, but got %v, with file sizes: %v",
			maxVariance,
			v,
			fileSizes,
		)
	}

}
