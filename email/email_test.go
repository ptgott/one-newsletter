package email

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	testkeypath  string = "mykey.pem"
	testcertpath string = "mycert.pem"
)

func TestUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		description   string
		input         string
		shouldBeError bool
	}{
		{
			description: "valid basic case",
			input: `type: basic
smtpServerAddress: 0.0.0.0:123
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: false,
		},
		{
			description: "valid sendgrid case",
			input: `type: sendgrid
apikey: abcdefgHIJKLMNOP012345679
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: false,
		},
		{
			description: "sendgrid case with space in the API key",
			input: `type: sendgrid
apikey: abcdefgHIJKL MNOP012345679
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description: "sendgrid case with tab in the API key",
			input: `type: sendgrid
apikey: abcdefgHIJKL	MNOP012345679
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description: "sendgrid case with a special character in the API key",
			input: `type: sendgrid
apikey: abcdefgHIJKL*MNOP012345679
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description: "sendgrid and no API key",
			input: `type: sendgrid
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description: "no to address",
			input: `type: basic
smtpServerAddress: 0.0.0.0:123
fromAddress: mynewsletter@example.com`,
			shouldBeError: true,
		},
		{
			description: "no type",
			input: `smtpServerAddress: 0.0.0.0:123
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description: "unrecognized type",
			input: `type: 123456
smtpServerAddress: 0.0.0.0:123
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description: "sendgrid type with no address",
			input: `type: sendgrid
fromAddress: mynewsletter@example.com
apikey: abcdefgHIJKLMNOP012345679
toAddress: recipient@example.com`,
			shouldBeError: false,
		},
		{
			description: "no from address",
			input: `type: basic
smtpServerAddress: 0.0.0.0:123
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description: "no server address",
			input: `type: basic
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com`,
			shouldBeError: true,
		},
		{
			description:   "not a map[string]string",
			input:         `[]`,
			shouldBeError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var uc UserConfig
			buf := bytes.NewBuffer([]byte(tc.input))
			dec := yaml.NewDecoder(buf)
			err := dec.Decode(&uc)
			if (err != nil) != tc.shouldBeError {
				t.Errorf(
					"%v: unexpected error status--wanted %v but got %v with error %v",
					tc.description,
					tc.shouldBeError,
					err != nil,
					err,
				)
			}
		})
	}
}

// smtpSender is shamelessly copied from Go's smtp package to wrap SMTP's
// requirement that each line end with CRLF
// See: https://github.com/golang/go/blob/289d34a465d46e5c5c07034f5d54afbfda06f5b9/src/net/smtp/smtp_test.go#L1028-L1034
type smtpSender struct {
	w io.Writer
}

// send is shamelessly copied from Go's smtp package to wrap SMTP's
// requirement that each line end with CRLF
// See: https://github.com/golang/go/blob/289d34a465d46e5c5c07034f5d54afbfda06f5b9/src/net/smtp/smtp_test.go#L1028-L1034
func (s smtpSender) send(f string) {
	s.w.Write([]byte(f + "\r\n"))
}

// serveFakeSMTP is copied from the test suite for the go smtp package. The
// difference is that it doesn't actually implement TLS
// https://github.com/golang/go/blob/289d34a465d46e5c5c07034f5d54afbfda06f5b9/src/net/smtp/smtp_test.go#L1036-L1062
func serveFakeSMTP(c net.Conn, dc *dataCatcher) error {
	send := smtpSender{c}.send
	// Matches the four-letter string that begins every SMTP client command
	commandPattern := regexp.MustCompile("^[A-Z]{4}")
	var afterData bool // Whether we've received a DATA or "." command
	// Read from the connection one line at a time. Since SMTP divides its own
	// messages into lines, this is a convenient way to process messages.
	send("220 service ready")
	s := bufio.NewScanner(c)
	for s.Scan() {
		match := commandPattern.FindString(s.Text())

		// The end of the email data
		if afterData && s.Text() == "." {
			afterData = false
			continue
		}

		// We received a DATA command, so match anything
		if afterData {
			dc.catch(s.Bytes()) // We're only matching the email body here
			continue
		}

		if match == "EHLO localhost" {
			send("250-127.0.0.1 ESMTP offers a warm hug of welcome")
			send("250 Ok")
			continue
		}

		if match == "DATA" {
			afterData = true
			send("354 send the mail data, end with .")
			send("250 Ok")
		}

		if match == "QUIT" {
			send("221 127.0.0.1 Service closing transmission channel")
			return nil
		}

		// Other commands
		if len(match) > 0 && !afterData {
			send("250 Ok")
			continue
		}
	}
	return s.Err()
}

// dataCatcher is used for inspecting the data sent to our fake SMTP server
type dataCatcher struct {
	data []byte
	mu   *sync.Mutex
}

// catch stores data in the dataCatcher for inspection later. It's safe for
// use in goroutines
func (dc *dataCatcher) catch(data []byte) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.data = append(dc.data, data...)
}

// TestSend is meant to test the minimal expected behavior of
// *SMTPClient.Send(), without setting up authentication or TLS
func TestSend(t *testing.T) {
	bodText := []byte("Hello this is my email body")
	bodHTML := []byte("<html><body>Hello this is my email body.</body></html>")

	rand.Seed(time.Now().UnixNano())
	p := rand.Intn(1000) + 1000 // quasi-random port > 1000

	uc := UserConfig{
		FromAddress:       "me@example.com",
		ToAddress:         "you@example.com",
		SMTPServerAddress: "localhost:" + strconv.Itoa(p),
		Type:              BasicType,
	}

	srv, err := net.Listen("tcp4", fmt.Sprintf("localhost:%v", p))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	d := &dataCatcher{
		data: []byte{},
		mu:   &sync.Mutex{},
	}

	// we only need to worry about one possible error
	errCh := make(chan error, 1)

	go func(l net.Listener, dc *dataCatcher, ec chan error) {
		conn, err := srv.Accept()
		if err != nil {
			errCh <- fmt.Errorf("problem accepting a new TCP connection: %v", err)
			return
		}
		serveFakeSMTP(conn, dc)
	}(srv, d, errCh)

	err = uc.SendNewsletter(bodText, bodHTML)

	if err != nil {
		t.Fatalf(
			"unexpected error when sending the email: %v",
			err,
		)
	}

	if len(errCh) > 0 {
		err := <-errCh
		t.Fatal(err)
	}

	if !strings.Contains(string(d.data), string(bodText)) {
		t.Error("the text/plain email body never reached the server")
	}

	if !strings.Contains(string(d.data), string(bodHTML)) {
		t.Error("the text/html email body never reached the server")
	}

}
