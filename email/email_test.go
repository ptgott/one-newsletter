package email

import (
	"bytes"
	"mime/multipart"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/ptgott/one-newsletter/smtptest"

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
			description: "valid case",
			input: `smtpServerAddress: smtp://0.0.0.0:123
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com
username: MyUser123
password: 123456-A_BCDE
`,
			shouldBeError: false,
		},
		// We should allow this because smtp:// is self evident
		{
			description: "no scheme",
			input: `smtpServerAddress: 0.0.0.0:123
fromAddress: mynewsletter@example.com
toAddress: recipient@example.com
username: MyUser123
password: 123456-A_BCDE
`,
			shouldBeError: false,
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

// TestSend is meant to test the minimal expected behavior of
// *SMTPClient.Send(), without setting up authentication or TLS
func TestSend(t *testing.T) {
	bodText := []byte("Hello this is my email body")
	bodHTML := []byte("<html><body>Hello this is my email body.</body></html>")

	k, c, err := smtptest.GenerateTLSFiles(t)
	if err != nil {
		t.Error(err)
	}
	srv := smtptest.NewInProcessServer(k, c)

	// The scheme isn't retunred by srv.Address(), so we add it here
	u, err := url.Parse("smtp://" + srv.Address())
	if err != nil {
		t.Error(err)
	}

	uc := UserConfig{
		FromAddress:          "me@example.com",
		ToAddress:            "you@example.com",
		SMTPServerHost:       u.Hostname(),
		SMTPServerPort:       u.Port(),
		UserName:             "myuser",
		Password:             "mypassword",
		SkipCertVerification: true, // since it's a self-signed cert
	}

	go func(srv *smtptest.InProcessServer) {
		srv.Start()
	}(srv)
	defer srv.Close()

	err = uc.SendNewsletter(bodText, bodHTML)
	if err != nil {
		t.Fatalf(
			"unexpected error when sending the email: %v",
			err,
		)
	}

	b, err := srv.RetrieveEmails(0)
	if err != nil {
		t.Error(err)
	}
	if len(b) != 1 {
		t.Fatalf("expected to have sent one email, but sent %v instead", len(b))
	}
	if !strings.Contains(b[0], string(bodText)) {
		t.Error("the text/plain email body never reached the server")
	}
	if !strings.Contains(b[0], string(bodHTML)) {
		t.Error("the text/html email body never reached the server")
	}

	bre := regexp.MustCompile(
		"Content-Type: multipart/alternative; boundary=(\\w+)",
	)
	m := bre.FindAllStringSubmatch(b[0], -1)
	if len(m) == 0 {
		t.Error("could not find the expected header with a boundary attribute")
	}

	bnd := m[0][1] // first capture group match, i.e., the boundary

	s := strings.SplitAfterN(b[0], "\r\n\r\n", 2)
	if len(s) < 2 {
		t.Errorf("expecting a blank line after the headers, but got none")
	}

	rdr := multipart.NewReader(
		bytes.NewBuffer([]byte(s[1])), // the email body, supposedly
		bnd,
	)

	expectedParts := map[string]struct{}{
		"text/plain": {},
		"text/html":  {},
	}
	var partMatches int
	for {
		p, err := rdr.NextPart()
		if err != nil {
			// For some reason, NextPart() returns an EOF as an
			// error _containing_ the string "EOF", not io.EOF
			// itself.
			if strings.Contains(err.Error(), "EOF") {
				break
			}
			t.Fatal(err)
		}
		if _, ok := expectedParts[p.Header.Get("Content-Type")]; !ok {
			t.Fatalf(
				"unexpected MIME type in header: %v",
				p.Header.Get("Content-Type"),
			)
		}
		partMatches++
	}
	if partMatches != len(expectedParts) {
		t.Errorf(
			"expected %v MIME parts but got %v",
			len(expectedParts),
			partMatches,
		)
	}

}

func TestCheckAndSetDefaults(t *testing.T) {
	cases := []struct {
		description        string
		input              UserConfig
		expected           UserConfig
		expectErrSubstring string
	}{
		{
			description: "straightforwarding valid case",
			input: UserConfig{
				SMTPServerHost:       "0.0.0.0",
				SMTPServerPort:       "25",
				FromAddress:          "mynewsletter@example.com",
				ToAddress:            "recipient@example.com",
				UserName:             "MyUser123",
				Password:             "123456-A_BCDE",
				SkipCertVerification: true,
			},
			expected: UserConfig{
				SMTPServerHost:       "0.0.0.0",
				SMTPServerPort:       "25",
				FromAddress:          "mynewsletter@example.com",
				ToAddress:            "recipient@example.com",
				UserName:             "MyUser123",
				Password:             "123456-A_BCDE",
				SkipCertVerification: true,
			},
		},
		{
			description: "no port",
			input: UserConfig{
				SMTPServerHost:       "0.0.0.0",
				FromAddress:          "mynewsletter@example.com",
				ToAddress:            "recipient@example.com",
				UserName:             "MyUser123",
				Password:             "123456-A_BCDE",
				SkipCertVerification: true,
			},
			expectErrSubstring: "port",
			expected:           UserConfig{},
		},
		{
			description: "no password",
			input: UserConfig{
				SMTPServerHost:       "0.0.0.0",
				SMTPServerPort:       "25",
				FromAddress:          "mynewsletter@example.com",
				ToAddress:            "recipient@example.com",
				UserName:             "MyUser123",
				SkipCertVerification: true,
			},
			expectErrSubstring: "password",
			expected:           UserConfig{},
		},
		{
			description: "no username",
			input: UserConfig{
				SMTPServerHost:       "0.0.0.0",
				SMTPServerPort:       "25",
				FromAddress:          "mynewsletter@example.com",
				ToAddress:            "recipient@example.com",
				Password:             "123456-A_BCDE",
				SkipCertVerification: true,
			},
			expectErrSubstring: "username",
			expected:           UserConfig{},
		},
		{
			description: "no to address",
			input: UserConfig{
				SMTPServerHost:       "0.0.0.0",
				SMTPServerPort:       "25",
				FromAddress:          "mynewsletter@example.com",
				UserName:             "MyUser123",
				Password:             "123456-A_BCDE",
				SkipCertVerification: true,
			},
			expectErrSubstring: "\"to\" address",
			expected:           UserConfig{},
		},
		{
			description: "no from address",
			input: UserConfig{
				SMTPServerHost:       "0.0.0.0",
				SMTPServerPort:       "25",
				ToAddress:            "recipient@example.com",
				UserName:             "MyUser123",
				Password:             "123456-A_BCDE",
				SkipCertVerification: true,
			},
			expectErrSubstring: "\"from\" address",
			expected:           UserConfig{},
		},
		{
			description: "no SMTP server host",
			input: UserConfig{
				SMTPServerPort:       "25",
				FromAddress:          "mynewsletter@example.com",
				ToAddress:            "recipient@example.com",
				UserName:             "MyUser123",
				Password:             "123456-A_BCDE",
				SkipCertVerification: true,
			},
			expectErrSubstring: "host",
			expected:           UserConfig{},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actual, err := c.input.CheckAndSetDefaults()
			if c.expectErrSubstring != "" && err == nil {
				t.Fatalf(
					"expected an error with substring %v but got nil",
					c.expectErrSubstring,
				)
			}
			if c.expectErrSubstring != "" &&
				!strings.Contains(err.Error(), c.expectErrSubstring) {
				t.Fatalf(
					"expected error with substring %v but got %v",
					c.expectErrSubstring,
					err,
				)
			}
			if c.expectErrSubstring == "" && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
			if !reflect.DeepEqual(actual, c.expected) {
				t.Fatalf("expected %+v but got %+v", c.expected, actual)
			}
		})
	}
}
