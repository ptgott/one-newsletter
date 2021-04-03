package email

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
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
	SMTPServerAddress string `yaml:"smtpServerAddress"`
	FromAddress       string `yaml:"fromAddress"`
	ToAddress         string `yaml:"toAddress"`
}

// isLocal determines whether a host is local by consulting its hostname. This
// assumes that the hostname is valid, and doesn't do any validation itself.
func isLocal(hostname string) bool {
	localHostnames := [...]string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
	}
	status := false
	for _, s := range localHostnames {
		if strings.Contains(hostname, s) {
			status = true
		}
	}
	return status
}

// UnmarshalYAML implements the yaml.Unmarshaler interface. Validation is
// performed here.
func (uc *UserConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return fmt.Errorf("can't parse the email config: %v", err)
	}

	ssa, ok := v["smtpServerAddress"]
	if !ok {
		return errors.New("email config must include the address of an SMTP server")
	}

	if !isLocal(ssa) {
		return fmt.Errorf("SMTP server address %v must be local", uc.SMTPServerAddress)
	}

	uc.SMTPServerAddress = ssa

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
	return nil
}

// SMTPClient handles interactions with the local SMTP server
type SMTPClient struct {
	fromAddress string
	toAddress   string
	smtpHost    string
	smtpPort    string
}

// NewSMTPClient validates user input and returns a Dialer
// that we can use to send actual email. Returns an error
// on validation failure.
func NewSMTPClient(uc *UserConfig) (*SMTPClient, error) {

	if uc.ToAddress == "" || uc.FromAddress == "" {
		return &SMTPClient{}, errors.New("must supply a \"to\" address and a \"from\" address")
	}

	if uc.SMTPServerAddress == "" {
		return &SMTPClient{}, errors.New("must supply an SMTP server address")
	}

	// Don't require the user to include a scheme. If we can't
	// find one, use one for SMTP.
	var ra string
	// Not handling the error since it only happens on compilation, which
	// won't fail since the regexp is constant.
	// https://github.com/golang/go/blob/b634f5d97a6e65f19057c00ed2095a1a872c7fa8/src/regexp/regexp.go#L560
	m, _ := regexp.Match(fmt.Sprintf("^%v", smtpScheme), []byte(uc.SMTPServerAddress))
	if m {
		ra = uc.SMTPServerAddress
	} else {
		ra = fmt.Sprintf("%v%v", smtpScheme, uc.SMTPServerAddress)
	}

	u, err := url.Parse(ra)

	if err != nil {
		return &SMTPClient{}, err
	}

	return &SMTPClient{
		fromAddress: uc.FromAddress,
		toAddress:   uc.ToAddress,
		smtpHost:    u.Hostname(),
		smtpPort:    u.Port(),
	}, nil

}

// Send sends the newsletter to the local SMTP server. Callers must suppply the
// newsletter as the `text/plain` MIME type in the asText param  and the
// `text/html` type in asHTML. A lack of an error means the message was
// received by the destination SMTP server.
func (sc *SMTPClient) SendNewsletter(asText, asHTML []byte) error {
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
	headerWriter.PrintfLine("From: Your Link Newsletter<%s>", sc.fromAddress)
	headerWriter.PrintfLine("To: <%s>", sc.toAddress)
	headerWriter.PrintfLine("Subject: New links to look at")
	headerWriter.PrintfLine("") // blank line before message body

	// Create the multipart/alternative RFC 2046 entity
	var ab bytes.Buffer
	altWriter := multipart.NewWriter(&ab)
	maw, _ := altWriter.CreatePart(
		map[string][]string{"Content-Type": {"multipart/alternative"}},
	)

	plainWriter := multipart.NewWriter(maw)
	pw, _ := plainWriter.CreatePart(
		map[string][]string{"Content-Type": {"text/plain"}},
	)
	_, err := pw.Write(asText)
	if err != nil {
		return err
	}

	htmlWriter := multipart.NewWriter(maw)
	hw, _ := htmlWriter.CreatePart(
		map[string][]string{"Content-Type": {"text/html"}},
	)
	_, err = hw.Write(asHTML)
	if err != nil {
		return err
	}

	msg.Write(ab.Bytes()) // add the multipart body to the email message
	msg.Flush()

	return smtp.SendMail(
		fmt.Sprintf("%v:%v", sc.smtpHost, sc.smtpPort),
		nil,
		sc.fromAddress,
		[]string{sc.toAddress},
		buf.Bytes(),
	)
}
