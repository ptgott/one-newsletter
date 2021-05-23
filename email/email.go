package email

import (
	"bufio"
	"bytes"
	"errors"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"regexp"
)

type localStatus int

const smtpScheme string = "smtp://"

type EmailClientType int

const (
	BasicType EmailClientType = iota
	SendGridType
)

// UserConfig represents config options provided the user. Not meant to be used
// directly for sending email without validation.
//
// Normally taking file paths as user input isn't great for testing, but we're
// accommodating the tls package which uses these.
// https://golang.org/pkg/crypto/tls/#LoadX509KeyPair
type UserConfig struct {
	SMTPServerAddress string // optional unless using basic
	FromAddress       string
	ToAddress         string
	ApiKey            string // optional unless using an SMTP API
	Type              EmailClientType
}

// UnmarshalYAML implements the yaml.Unmarshaler interface. Validation is
// performed here.
func (uc *UserConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	recognizedTypes := map[string]EmailClientType{
		"basic":    BasicType,
		"sendgrid": SendGridType,
	}

	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return errors.New("the email config must be an object")
	}

	tp, ok := v["type"]
	if !ok {
		return errors.New("the email config must specify a type")
	}

	if _, ok := recognizedTypes[tp]; !ok {
		return errors.New("unrecognized email client type: " + tp)
	}

	uc.Type = recognizedTypes[tp]

	ak, ok := v["apikey"]
	if !ok && uc.Type != BasicType {
		return errors.New("must provide an apikey config option for SMTP APIs")
	}

	if !regexp.MustCompile("^[A-Za-z0-9]+$").MatchString(ak) && uc.Type != BasicType {
		return errors.New("the API key must include only alphanumeric characters, with no whitespace")
	}

	uc.ApiKey = ak // empty string if not provided

	ssa, ok := v["smtpServerAddress"]
	// sendgrid has a fixed address
	if !ok && tp != "sendgrid" {
		return errors.New("email config must include the address of an SMTP server")
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

// SendNewsletter sends the newsletter to the SMTP server. Callers must supply the
// newsletter as the `text/plain` MIME type in the asText param  and the
// `text/html` type in asHTML. A lack of an error means the message was
// received by the destination SMTP server.
func (uc UserConfig) SendNewsletter(asText, asHTML []byte) error {
	sendGridHost := "smtp.sendgrid.net"
	var addr string
	if uc.Type == SendGridType {
		// https://sendgrid.com/docs/for-developers/sending-email/integrating-with-the-smtp-api/#smtp-ports
		addr = sendGridHost + ":587"
	} else {
		addr = uc.SMTPServerAddress
	}

	var auth smtp.Auth

	if uc.Type == SendGridType {
		// Using username "apikey" with the user's API key as the password
		// See:
		// https://sendgrid.com/docs/for-developers/sending-email/integrating-with-the-smtp-api/
		auth = smtp.PlainAuth("", "apikey", uc.ApiKey, sendGridHost)
	} else {
		auth = nil
	}

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
	pw.Write(asText)

	htmlWriter := multipart.NewWriter(maw)
	hw, _ := htmlWriter.CreatePart(
		map[string][]string{"Content-Type": {"text/html"}},
	)
	hw.Write(asHTML)

	msg.Write(ab.Bytes()) // add the multipart body to the email message
	msg.Flush()

	return smtp.SendMail(
		addr,
		auth,
		uc.FromAddress,
		[]string{uc.ToAddress},
		buf.Bytes(),
	)
}
