package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	gomail "gopkg.in/gomail.v2"
)

const smtpScheme string = "smtp://"

// UserConfig represents config options provided by
// the user. Not meant to be used directly for sending
// email without validation.
//
// Normally taking file paths as user input isn't great
// for testing, but we're accommodating the tls package,
// which uses these.
// https://golang.org/pkg/crypto/tls/#LoadX509KeyPair
type UserConfig struct {
	RelayAddress string
	Key          []byte // PEM-encoded TLS key
	Cert         []byte // PEM-encoded TLS cert
	Username     string
	Password     string
	FromAddress  string
	ToAddress    string
}

// NewSMTPClient validates user input and returns a Dialer
// that we can use to send actual email. Returns an error
// on validation failure.
func NewSMTPClient(uc UserConfig) (*SMTPClient, error) {

	if uc.Password == "" || uc.Username == "" {
		return &SMTPClient{}, errors.New("must supply a username and password")
	}

	if uc.ToAddress == "" || uc.FromAddress == "" {
		return &SMTPClient{}, errors.New("must supply a \"to\" address and a \"from\" address")
	}

	// Don't require the user to include a scheme. If we can't
	// find one, use one for SMTP.
	var ra string
	// Not handling the error since it only happens on compilation, which
	// won't fail since the regexp is constant.
	// https://github.com/golang/go/blob/b634f5d97a6e65f19057c00ed2095a1a872c7fa8/src/regexp/regexp.go#L560
	m, _ := regexp.Match(fmt.Sprintf("^%v", smtpScheme), []byte(uc.RelayAddress))
	if m {
		ra = uc.RelayAddress
	} else {
		ra = fmt.Sprintf("%v%v", smtpScheme, uc.RelayAddress)
	}

	u, err := url.Parse(ra)

	if err != nil {
		return &SMTPClient{}, err
	}

	p, err := strconv.Atoi(u.Port())

	if err != nil {
		return &SMTPClient{}, err
	}

	cert, err := tls.X509KeyPair(uc.Cert, uc.Key)

	if err != nil {
		return &SMTPClient{}, err
	}

	tlsc := tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
	}

	return &SMTPClient{
		FromAddress: uc.FromAddress,
		ToAddress:   uc.ToAddress,
		dialer: gomail.Dialer{
			Host:      u.Hostname(),
			Port:      p,
			Username:  uc.Username,
			Password:  uc.Password,
			SSL:       true,
			TLSConfig: &tlsc,
		},
	}, nil

}

// SMTPClient handles interactions with the local SMTP server
type SMTPClient struct {
	dialer      gomail.Dialer
	FromAddress string
	ToAddress   string
}

// Send sends the HTML message in body to the local SMTP server. A lack of an
// error means the message was received by the destination SMTP server.
func (sc *SMTPClient) Send(body string) error {

	m := gomail.NewMessage()
	m.SetHeader("From", sc.FromAddress)
	m.SetHeader("To", sc.ToAddress)
	m.SetHeader("Subject", "The latest from DivToNewsletter")
	m.SetBody("text/html", body)

	return sc.dialer.DialAndSend(m)
}
