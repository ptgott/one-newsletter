package e2e

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/ptgott/one-newsletter/smtptest"
)

const (
	tempDirPathName = "tempTestDir"
)

// testEnvironmentConfig exposes options that should be available and
// perhaps changeable when spinning up a test environment. While they
// may not vary between tests, they shouldn't be buried inside
// functions.
type testEnvironmentConfig struct {
	numHTTPServers int // How many mock web publications to spin up
	numLinks       int //How many links this application should be able to scrape from
	// each web publication
}

// testEnvironment manages all dependencies required to simulate a "real"
// environment and run the e2e tests. Callers should create this via
// startTestEnvironment.
type testEnvironment struct {
	*testServerGroup
	SMTPServer  smtptest.Server
	tempDirPath string // must be populated programmatically
}

// startTestEnvironment spins up dependencies (including possibly in child
// processes). Callers should defer a call to tearDown.
//
// Note that if startTestEnvironment fails, it will return an error along with
// whatever shreds of a test environment we've set up so far so you can tear
// it down (i.e., it won't just be the zero value)
func startTestEnvironment(t *testing.T, c testEnvironmentConfig) (*testEnvironment, error) {
	te := &testEnvironment{}

	p, err := os.MkdirTemp(".", tempDirPathName)
	// Ignore errors due to the fact that the directory already exists
	if err != nil && !errors.Is(err, os.ErrExist) {
		// Shouldn't happen
		return te, fmt.Errorf("could not create the test storage directory: %w", err)
	}

	te.tempDirPath = p

	key, cert, err := smtptest.GenerateTLSFiles(t)
	if err != nil {
		return nil, err
	}
	ts := smtptest.NewInProcessServer(key, cert)

	te.SMTPServer = ts

	go ts.Start()

	sg := startTestServerGroup(c.numHTTPServers, c.numLinks)

	te.testServerGroup = sg

	return te, nil
}

// tearDown returns the testEnvironment to its state prior to start. Designed
// to call with defer
func (te *testEnvironment) tearDown() {
	if te.SMTPServer != nil {
		te.SMTPServer.Close()
	}

	if te.testServerGroup != nil {
		te.testServerGroup.close()
	}

	// This error will be nil if the path doesn't exist. See:
	// https://golang.org/pkg/os/#RemoveAll
	err := os.RemoveAll(te.tempDirPath)

	// We're not expecting this to return an error since it's designed to call with
	// defer. Instead we panic, and hopefully we can prevent any panic-causing
	// error from happening again.
	if err != nil {
		panic(fmt.Sprintf("can't delete the test storage directory: %v", err))
	}
}
