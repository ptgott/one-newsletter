package e2e

import (
	"divnews/testutil"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

var (
	mailHogPath string // path to the MailHog executable taken from user config
)

func TestMain(m *testing.M) {
	j, err := os.ReadFile("../e2e_config.json")
	if err != nil {
		panic(fmt.Sprintf("can't open the e2e config file: %v", err))
	}
	var opts map[string]string
	err = json.Unmarshal(j, &opts)
	if err != nil {
		panic(fmt.Sprintf("can't parse the e2e test config as json: %v", err))
	}
	mh, ok := opts["mailhog_path"]
	if !ok {
		panic("the e2e config file must specify a mailHogPath")
	}
	mailHogPath = mh
	os.Exit(m.Run())
}

func TestNewsletterEmail(t *testing.T) {
	stopIntervalS := 6
	pollIntervalS := 4
	epubs := 3
	linksPerPub := 5
	testenv, err := startTestEnvironment(testEnvironmentConfig{
		numHTTPServers: epubs,
		numLinks:       linksPerPub,
		mailHogPath:    mailHogPath,
	})

	defer testenv.tearDown()

	if err != nil {
		t.Fatalf("error starting test environment: %v", err)
	}

	err = createAppConfig(
		fmt.Sprintf("%v/%v", testenv.tempDirPath, "config.yaml"),
		appConfigOptions{
			RelayAddress:   testenv.testSMTPServer.smtpAddress(),
			SSLKey:         testutil.TestKey,
			SSLCert:        testutil.TestCert,
			LinkSourceURLs: testenv.urls(),
			StorageDir:     testenv.tempDirPath,
			PollInterval:   fmt.Sprintf("%vs", pollIntervalS),
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Build and run the application from the entrypoint with our new config
	cmd := exec.Command(
		"go",
		"run",
		"../main.go",
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
	)

	// create a pipe to collect logs from the application
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("couldn't create a pipe to the application: %v", err)
	}

	if err = cmd.Start(); err != nil {
		t.Fatalf("couldn't start the app: %v", err)
	}

	time.Sleep(time.Duration(stopIntervalS) * time.Second)
	b, err := io.ReadAll(stderr)
	if err != nil {
		t.Fatalf("couldn't read from the pipe to the application: %v", err)
	}

	os.Stdout.Write(b)
	err = cmd.Process.Signal(os.Interrupt)

	// At this point you need to find the process and kill it manually.
	// This messes up the test, so we panic.
	if err != nil {
		panic(fmt.Sprintf("pid %v could not be interrupted", cmd.Process.Pid))
	}

	ems, err := testenv.retrieveEmails()

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

	// TODO: Make a test assertion about the content of an email
	// TODO: Make a test assertion about the database
}
