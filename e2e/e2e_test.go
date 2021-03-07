package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
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

func TestNewsletterEmail(t *testing.T) {
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
		},
	)
	if err != nil {
		panic(fmt.Sprintf("can't create the app config: %v", err))
	}

	// Build and run the application from the entrypoint with our new config
	cmd := exec.Command(
		appPath,
		fmt.Sprintf("-config=%v/%v", testenv.tempDirPath, "config.yaml"),
	)

	// create a pipe to collect logs from the application
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("couldn't create a pipe to the application: %v", err)
	}

	// Read repeatedly from stderr to get logs from the application. Need to do
	// this in the background, rather than in one big slurp after Wait(), since
	// otherwise the OS closes the read side of the pipe.
	go func(f *os.File) {
		for {
			// ReadAll should be fine here since we're continuously reading
			// from a pipe.
			// Skipping errors since we might get a successful read on the next
			// iteration.
			b, _ := io.ReadAll(f)
			os.Stdout.Write(b)
		}
	}(stderr.(*os.File))

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

	// TODO: Make a test assertion about the content of an email (e.g.,
	// that it includes the required email headers as well as the multipart
	// entities)
	// TODO: Make a test assertion about the database
}
