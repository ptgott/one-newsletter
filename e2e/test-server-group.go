package e2e

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"sync"
)

type mockArticleListing struct {
	Caption string
	URL     string
}

const linkSiteTmpl string = `<!doctype html>
<html>
<body>
<h1>Welcome to my cool newspaper!</h1>
<h2>Daily headlines:<h2>
<ul>
{{ range . }}
<li>
<p>.Caption</p>
<a href="{{.URL}}">Check this out</a>
</li>
{{ end }}
</ul>
</body>
</html>
`

// generateLinkMenuEndpoint creates an http.HandlerFunc that returns HTML to
// clients simulating an online publication with a list of numLinks links.
// Determining the number of links from outside the function lets us use that
// number in test assertions.
func generateLinkMenuEndpoint(numLinks int) http.HandlerFunc {
	listings := make([]mockArticleListing, numLinks, numLinks)
	for i := range listings {
		listings[i] = mockArticleListing{
			Caption: fmt.Sprintf("Article %v", i),
			URL:     fmt.Sprintf("https://www.example.com/articles/%v", i),
		}
	}

	tmpl, err := template.New("listings").Parse(linkSiteTmpl)
	if err != nil {
		// This is an error with the test suite, not the application
		panic(fmt.Sprintf("error parsing the link site template: %v", err))
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, listings)
	if err != nil {
		// This is an error with the test suite, not the application
		panic(fmt.Sprintf("error executing the link site template: %v", err))
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write(buf.Bytes())
	})
}

// startTestServerGroup spins up numServers in-process HTTP servers
// for simulating an e-publication to scrape. Each includes numLinks links
// available to scrape at the root URL path.
//
// Note that callers are responsible for closing each test server!
func startTestServerGroup(numServers int, numLinks int) *testServerGroup {
	servs := make([]*httptest.Server, numServers, numServers)
	for i := range servs {
		sm := http.NewServeMux()
		sm.HandleFunc("/", generateLinkMenuEndpoint(numLinks))
		servs[i] = httptest.NewServer(sm)
	}

	return &testServerGroup{
		servers: servs,
	}
}

// testServerGroup simulates a set of scrapable websites for e2e testing
type testServerGroup struct {
	servers []*httptest.Server
}

// close gracefully shuts down all servers in the TestServerGroup
func (tsg *testServerGroup) close() {
	// If there are no servers to close, do nothing
	if len(tsg.servers) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(tsg.servers))

	for i := range tsg.servers {
		go func(ts *httptest.Server, wg *sync.WaitGroup) {
			ts.Close()
			wg.Done()
		}(tsg.servers[i], &wg)
	}
	wg.Wait()
}

// urls returns (in string form) the URL of each server in the
// server group. Used for querying (or configuring queries for)
// the test servers.
func (tsg *testServerGroup) urls() []string {
	u := make([]string, len(tsg.servers), len(tsg.servers))
	for i := range tsg.servers {
		u[i] = tsg.servers[i].URL
	}
	return u
}
