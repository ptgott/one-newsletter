package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	mhdata "github.com/mailhog/data"
)

// MailHog contains information used for managing a MailHog server.
//
// Implements testSMTPServer.
type MailHog struct {
	// path to the mailHog executable
	mailHogPath string
	// port of the running MailHog SMTP endpoint (which is always local)
	smtpPort int
	// port for the MailHog API endpoint (which is always local)
	apiPort int
	// proc is used for managing the MailHog process
	proc *os.Process
}

// start runs the MailHog executable and returns an error if there were
// any problems starting the command. Note that this will not return an error
// if MailHog fails after starting--use checkHealth for that.
func (mh *MailHog) start() error {
	if mh.apiPort == 0 || mh.smtpPort == 0 {
		return errors.New("must specify an API and SMTP port for MailHog")
	}

	_, err := os.Lstat(mh.mailHogPath)

	if err != nil {
		return fmt.Errorf("can't find the MailHog executable: %v", err)
	}

	c := exec.Command(mh.mailHogPath)

	// Note that we're not using an authfile here to authenticate to MailHog.
	// This is only used to authenticate to the HTTP API, and isn't used for the
	// SMTP server.
	// https://github.com/mailhog/MailHog/blob/master/docs/Auth.md#authentication
	//
	// Set up ports/other config
	// https://github.com/mailhog/MailHog/blob/0441dd494b03c9255a9b8e90e3458ebb115eacff/docs/CONFIG.md
	c.Env = append(c.Env, fmt.Sprintf("MH_API_BIND_ADDR=0.0.0.0:%v", mh.apiPort))
	c.Env = append(c.Env, fmt.Sprintf("MH_SMTP_BIND_ADDR=0.0.0.0:%v", mh.smtpPort))
	err = c.Start()

	if err != nil {
		return fmt.Errorf("could not start MailHog: %v", err)
	}

	mh.proc = c.Process

	return nil
}

// close attempts to gracefully terminate the MailHog process and, failing
// that, kill it abruptly.
func (mh *MailHog) close() {
	// If the process isn't running, don't worry about attempting to exit it
	if mh.proc == nil {
		return
	}

	err := mh.proc.Signal(os.Interrupt)
	if err != nil {
		err = mh.proc.Kill()
		if err != nil {
			// we don't want to return an error here--panic so the user can
			// chase down the process manually.
			panic(fmt.Sprintf("could not terminate process %v: %v", mh.proc.Pid, err))
		}
	}
}

// messageResult contains the response body from the MailHog messages API
// endpoint. It's defined this way within the MailHog source, but isn't
// actually exported.
//
// https://github.com/mailhog/MailHog-Server/blob/50f74a1aa2991b96313144d1ac718ce4d6739dfd/api/v2.go#L72-L77
type messagesResult struct {
	Total int              `json:"total"`
	Count int              `json:"count"`
	Start int              `json:"start"`
	Items []mhdata.Message `json:"items"`
}

// retrieveEmails returns the bodies of the emails currently held by MailHog so
// we can inspect them.
//
// Uses the API documented here:
// https://github.com/mailhog/MailHog/blob/0441dd494b03c9255a9b8e90e3458ebb115eacff/docs/APIv2/swagger-2.0.yaml
func (mh *MailHog) retrieveEmails() ([]string, error) {
	msgPath := "/api/v2/messages"

	resp, err := http.Get(fmt.Sprintf("http://0.0.0.0:%v%v", mh.apiPort, msgPath))

	if err != nil {
		return []string{}, fmt.Errorf(
			"can't retrieve emails from the local MailHog server: %v", err,
		)
	}
	var buf bytes.Buffer

	if resp.StatusCode == 401 {
		return []string{}, fmt.Errorf(
			"got a 401 Unauthorized from the MailHog API--authenticate via %v",
			resp.Header.Get("WWW-Authenticate"),
		)
	}

	// The request was not a success. See:
	// https://github.com/mailhog/MailHog/blob/0441dd494b03c9255a9b8e90e3458ebb115eacff/docs/APIv2/swagger-2.0.yaml#L30
	if resp.StatusCode != 200 {
		return []string{}, fmt.Errorf(
			"got non-200 status code of %v from the MailHog API",
			resp.Status,
		)
	}

	n, err := buf.ReadFrom(resp.Body)

	if n == 0 {
		return []string{}, errors.New(
			"got an empty response body from the MailHog server",
		)
	}

	if err != nil {
		return []string{}, fmt.Errorf(
			"can't read emails from the local MailHog server: %v", err,
		)
	}
	var m messagesResult

	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		return []string{}, fmt.Errorf(
			"can't read the API response as JSON: %v", err,
		)
	}

	s := make([]string, len(m.Items), len(m.Items))
	for i := range m.Items {
		s[i] = m.Items[i].Content.Body
	}
	return s, nil
}

// smtpAddress retrieves the address of the SMTP server
func (mh *MailHog) smtpAddress() string {
	return fmt.Sprintf("0.0.0.0:%v", mh.smtpPort)
}
