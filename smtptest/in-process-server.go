package smtptest

import (
	"crypto/tls"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-units"
	"github.com/emersion/go-smtp"
)

// messageData includes the body content and created timestamp for an email
// message, allowing us to inspect message bodies before/after a timestamp
// for correctness.
type messageData struct {
	created time.Time
	body    string
}

// Backend implements smtp.Backend. It's a thin authentication wrapper
// for an InMemoryEmailStore.
type Backend struct {
	*InMemoryEmailStore
}

// Login implements smtp.Backend. Any username/password is fine, since we
// don't want to couple this with specific test configurations.
func (be *Backend) Login(_ *smtp.ConnectionState, username string, password string) (smtp.Session, error) {
	if username != "" && password != "" {
		return be.InMemoryEmailStore, nil
	}
	return nil, errors.New("no username or password provided")
}

// AnonymouseLogin implements smtp.Backend. Not supported since we want to
// enforce AUTH.
func (be *Backend) AnonymousLogin(_ *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthUnsupported
}

// InMemoryEmailStore retains email bodies in memory for comparison against
// a test's expected output. Implements smtp.Session.
// Designed to be goroutine safe since we don't know how many goroutines will
// be hitting the server at once.
type InMemoryEmailStore struct {
	mu       *sync.Mutex
	messages []messageData
}

// Reset implements smtp.Session. No-op here.
func (es *InMemoryEmailStore) Reset() { return }

// Logout implements smtp.Session. No-op here.
func (es *InMemoryEmailStore) Logout() error { return nil }

// Mail implements smtp.Session. No-op here.
func (es *InMemoryEmailStore) Mail(_ string, _ smtp.MailOptions) error { return nil }

// Rcpt implements smtp.Session. No-op here.
func (es *InMemoryEmailStore) Rcpt(to string) error { return nil }

// Rcpt implements smtp.Session. Stores the email data in memory for retrieval
// at the end of the test.
func (es *InMemoryEmailStore) Data(r io.Reader) error {
	// doubtful we'll get an email this big, but we need a limit
	var maxEmailSize int64 = 100 * units.MiB
	buf, err := io.ReadAll(io.LimitReader(r, maxEmailSize))
	if err != nil {
		return err
	}

	str := &strings.Builder{}
	if _, err := str.Write(buf); err != nil {
		return err
	}
	es.saveEmail(str.String())
	return nil
}

// InProcessServer is an SMTPServer that runs in the same process as the
// test suite, letting us inspect sent emails. You must initialize this
// via NewInProcessServer
type InProcessServer struct {
	*smtp.Server
	// We're also using this as an smtp.Session, i.e., the BAckend of the
	// *smtp.Server. This is kind of gross, but allows us to access the
	// *InmemoryEmailStore. Otherwise, we're stuck with *smtp.Server.Backend,
	// which just leaves us with the Backend interface methods.
	*InMemoryEmailStore
}

// NewInProcessServer creates an InProcessServer, including configuring
// its SMTP server to store incoming messages in memory. Must provide
// the paths to the key and cert used for TLS. The cert must be a
// root cert.
func NewInProcessServer(keypath string, certpath string) *InProcessServer {
	is := &InMemoryEmailStore{
		mu:       &sync.Mutex{},
		messages: []messageData{},
	}

	srv := smtp.NewServer(&Backend{
		is,
	})

	srv.Addr = ":2526" // arbitrary
	srv.Domain = "localhost"
	srv.AllowInsecureAuth = false // need AUTH here
	srv.AuthDisabled = false      // need AUTH here
	// Strict is undocumented, but it looks like it enforces <address> syntax
	// in messages:
	// https://github.com/emersion/go-smtp/blob/f92bf7f1a25777bcdaa28a142b1cd1a54b74c8f4/conn.go#L321-L325
	srv.Strict = true

	cert, err := tls.LoadX509KeyPair(certpath, keypath)

	// No way to carry on without a cert, so we panic. We're in a test
	// suite, so this should be fine.
	if err != nil {
		panic(err)
	}

	srv.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	ip := &InProcessServer{
		srv,
		is,
	}

	return ip
}

// saveEmail stores the email body in memory along with a timestamp created
// just prior to saving
func (es *InMemoryEmailStore) saveEmail(bod string) {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.messages = append(es.messages, messageData{
		created: time.Now(),
		body:    bod,
	})

}

// Start starts the test server. Blocking.
func (is *InProcessServer) Start() error {
	// Not using ListenAndServeTLS--the client should upgrade the connection
	// to TLS
	return is.Server.ListenAndServe()
}

// Close shuts down the test server daemon. You must initialize a new
// InProcessServer instead of restarting this one.
func (is *InProcessServer) Close() {
	is.Server.Close()
}

// RetrieveEmails returns a slice of all message bodies (as strings)
// sent after epoch nanoseconds t
// Satisfies smtptest.Server but isn't expected to return an error.
func (es *InMemoryEmailStore) RetrieveEmails(t int64) ([]string, error) {
	r := make([]string, 0, len(es.messages))
	for _, m := range es.messages {
		if m.created.UnixNano() >= t {
			r = append(r, m.body)
		}
	}
	return r, nil
}

// Address returns the host:port of the test SMTP server.
func (is *InProcessServer) Address() string {
	return is.Server.Domain + is.Server.Addr
}
