package email

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

type localStatus int

const smtpScheme string = "smtp://"

// UserConfig represents config options provided the user. Not meant to be used
// directly for sending email without validation.
//
// Normally taking file paths as user input isn't great for testing, but we're
// accommodating the tls package which uses these.
// https://golang.org/pkg/crypto/tls/#LoadX509KeyPair
type UserConfig struct {
	SMTPServerHost string
	SMTPServerPort string
	FromAddress    string
	ToAddress      string
	UserName       string
	Password       string
	// Should only be used during testing. We can simulate all aspects of TLS
	// in a test environment but certification verification, since any cert used
	// by a test server would need to be self signed.
	SkipCertVerification bool
}

// UnmarshalYAML implements the yaml.Unmarshaler interface. Validation is
// performed here.
func (uc *UserConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {

	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return errors.New("the email config must be an object")
	}

	// This option must not be used outside tests, so we don't enforce it.
	scv, _ := v["skipCertVerification"]
	if scv == "true" {
		uc.SkipCertVerification = true
		log.Warn().Msg(
			"SKIPPING TLS CERTIFICATE VERIFICATION. THIS SHOULD BE A TEST ENVIRONMENT. YOU HAVE BEEN WARNED",
		)
	}

	ssa, ok := v["smtpServerAddress"]
	if !ok {
		return errors.New("email config must include the address of an SMTP server")
	}

	// We allow users to omit the scheme, since smtpServerAddress is only for
	// one protocol.
	if !strings.HasPrefix(ssa, "smtp://") {
		ssa = "smtp://" + ssa
	}

	u, err := url.Parse(ssa)

	if err != nil {
		return errors.New("the SMTP server address is not a valid URL: " + err.Error())
	}

	pr := u.Port()
	if pr == "" {
		return errors.New("the SMTP server address must include a port")
	}

	uc.SMTPServerHost = u.Hostname()
	uc.SMTPServerPort = pr

	fa, ok := v["fromAddress"]
	if !ok {
		return errors.New("email config must include a \"from\" adddress for sending email")
	}
	uc.FromAddress = fa

	ta, ok := v["toAddress"]
	if !ok {
		return errors.New("email confic must include a \"to\" address for sending email")
	}
	uc.ToAddress = ta

	un, ok := v["username"]
	if !ok {
		return errors.New("email config must include a username for the SMTP relay server or MTA")
	}
	uc.UserName = un

	pw, ok := v["password"]
	if !ok {
		return errors.New("email config must include a password for the SMTP relay server or MTA")
	}
	uc.Password = pw
	return nil
}

// SendNewsletter sends the newsletter to the SMTP server. Callers must supply the
// newsletter as the `text/plain` MIME type in the asText param  and the
// `text/html` type in asHTML. A lack of an error means the message was
// received by the destination SMTP server.
func (uc UserConfig) SendNewsletter(asText, asHTML []byte) error {

	auth := smtp.PlainAuth("", uc.UserName, uc.Password, uc.SMTPServerHost)

	// Write the email body. It will have the following MIME entities.
	// For more information see:
	// - https://tools.ietf.org/html/rfc2045 (MIME headers)
	// - https://tools.ietf.org/html/rfc2046#section-5 (MIME entity bodies)
	//
	//  |- multipart/alternative
	//  |  |- text/plain
	//  |  |- text/html
	//
	// Note that as per RFC 2046, we're putting the `text/html` entity
	// last within the "multipart/alternative" entity since it's the best
	// representation of the document. Servers can use the `text/plain`
	// entity as well if they need to.

	// Write the RFC 822 message headers. We need to do this manually. See:
	// https://golang.org/pkg/net/smtp/#SendMail
	var buf bytes.Buffer
	msg := bufio.NewWriter(&buf)
	headerWriter := textproto.NewWriter(msg)
	headerWriter.PrintfLine("From: Your Link Newsletter<%s>", uc.FromAddress)
	headerWriter.PrintfLine("To: <%s>", uc.ToAddress)
	headerWriter.PrintfLine("Subject: New links to look at")

	// Create the multipart/alternative RFC 2046 entity
	var ab bytes.Buffer
	altWriter := multipart.NewWriter(&ab)

	// Write the multipart/alternative boundary to a Content-Type header before
	// we write the message body
	headerWriter.PrintfLine(
		"Content-Type: multipart/alternative; boundary=%v",
		altWriter.Boundary(),
	)
	headerWriter.PrintfLine("") // blank line before message body

	pw, _ := altWriter.CreatePart(
		map[string][]string{"Content-Type": {"text/plain"}},
	)
	pw.Write(asText)

	hw, _ := altWriter.CreatePart(
		map[string][]string{"Content-Type": {"text/html"}},
	)
	hw.Write(asHTML)

	msg.Write(ab.Bytes()) // add the multipart body to the email message
	msg.Flush()

	// Send the email. This is copied with minor adjustments from smtp.SendMail
	// See: https://golang.org/src/net/smtp/smtp.go?s=9381:9459#L313

	// Connect to the remote SMTP server.
	c, err := smtp.Dial(uc.SMTPServerHost + ":" + uc.SMTPServerPort)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to the remote SMTP server")
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{
			ServerName: uc.SMTPServerHost,
			// For testing only, since we can't verify the self-signed cert used
			// by our test server.
			InsecureSkipVerify: uc.SkipCertVerification,
		}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	} else {
		return errors.New("SMTP server does not support STARTTLS")
	}

	if ok, _ := c.Extension("AUTH"); !ok {
		return errors.New("SMTP server doesn't support AUTH")
	}
	if err = c.Auth(auth); err != nil {
		return err
	}

	if err := c.Mail(uc.FromAddress); err != nil {
		return err
	}

	// Just using one recipient
	if err := c.Rcpt(uc.ToAddress); err != nil {
		return err
	}

	wc, err := c.Data()
	if err != nil {
		return err
	}
	_, err = wc.Write(buf.Bytes())
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	err = c.Quit()
	if err != nil {
		return err
	}
	return nil
}
