package e2e

// testSMTPServer contains state information for an SMTP server running as a
// separate process. The SMTP server should be able to return the payloads
// of messages sent to it during the test suite. The server is meant to start
// during a test (or test suite) and stop right after.
type testSMTPServer interface {
	// start launches the server as a separate process and returns an error
	// if this fails. Retry behavior is left to the caller. start should also
	// set up any resources, such as local files, required to run the server.
	start() error
	// close terminates the serve process and any required resources. While
	// this is designed not to return an error so it's easier to use with defer,
	// implementations should log failures to close so the test operator can
	// chase down rogue server processes.
	close()
	// retrieveEmails returns the payloads of all email messages sent to the
	// server during the test/suite.
	retrieveEmails() ([]string, error)
	// smtpAddress returns the address of the server. Getting this could vary
	// between implementations, so we make it a method.
	smtpAddress() string
}
