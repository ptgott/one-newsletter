package e2e

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
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
<p>{{.Caption}}</p>
<a href="{{.URL}}">Check this out</a>
</li>
{{ end }}
</ul>
</body>
</html>
`

// ServeHTTP implements http.Handler.
// It returns HTML to clients simulating an online publication with a list of
// links, and always returns the fakeEPublication's most recently updated
// batch of links.
func (fp fakeEPublication) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	tmpl, err := template.New("listings").Parse(linkSiteTmpl)
	if err != nil {
		// This is an error with the test suite, not the application
		panic(fmt.Sprintf("error parsing the link site template: %v", err))
	}

	if fp.numLinks == 0 || len(fp.updates) == 0 {
		panic("the e-publication has no content to generate")
	}

	// sort updates by timestamp and grab the most recent one to use for
	// generating HTML
	listings := make([]int, 0, fp.numLinks)
	for k := range fp.updates {
		listings = append(listings, int(k))
	}
	listings = sort.IntSlice(listings)
	latest := listings[len(listings)-1]

	err = tmpl.Execute(rw, fp.updates[int64(latest)])
	if err != nil {
		// This is an error with the test suite, not the application
		panic(fmt.Sprintf("error executing the link site template: %v", err))
	}
}

// createContent generates a brand new linkUpdate using attributes of the
// fakeEPublication
func (fp fakeEPublication) createContent() []mockArticleListing {
	listings := make([]mockArticleListing, fp.numLinks, fp.numLinks)

	for i := range listings {
		listings[i] = fp.newMockArticleListing()
	}

	return listings

}

func (fp fakeEPublication) newMockArticleListing() mockArticleListing {
	u := uuid.NewString()
	return mockArticleListing{
		Caption: fmt.Sprintf("Article %v", u),
		URL: fmt.Sprintf(
			"https://%v.example.com/articles/%v",
			fp.id,
			u,
		),
	}
}

// refreshContent returns a new linkUpdate that swaps out toReplace links with
// new ones
func (fp fakeEPublication) refreshContent(mps []mockArticleListing, toReplace int) []mockArticleListing {
	li := make([]mockArticleListing, len(mps), len(mps))
	// cleanly copy mps
	for i := range mps {
		li[i] = mps[i]
	}
	if toReplace > len(mps) {
		panic("trying to refresh more content than is available")
	}
	rand.Seed(time.Now().UnixNano())

	// Shuffle the listings so we can pick the first n to replace
	rand.Shuffle(len(li), func(i, j int) {
		li[i], li[j] = li[j], li[i]
	})

	for k := 0; k < toReplace; k++ {
		li[k] = fp.newMockArticleListing()
	}

	return li
}

// startTestServerGroup spins up numServers in-process HTTP servers
// for simulating an e-publication to scrape. Each includes numLinks links
// available to scrape at the root URL path.
//
// Note that callers are responsible for closing each test server!
func startTestServerGroup(numServers int, numLinks int) *testServerGroup {
	if numLinks <= 0 || numServers <= 0 {
		panic("numLinks and numServers must be > 0")
	}

	servs := make([]fakeEPublication, numServers, numServers)
	for i := range servs {
		servs[i] = fakeEPublication{
			numLinks: numLinks,
			id:       uuid.NewString(),
			updates:  make(map[int64][]mockArticleListing),
		}
		// the first update
		servs[i].updates[time.Now().Unix()] = servs[i].createContent()
		servs[i].server = httptest.NewServer(servs[i])
	}

	return &testServerGroup{
		sites: servs,
	}
}

// testServerGroup simulates a set of scrapable websites for e2e testing
type testServerGroup struct {
	sites []fakeEPublication
}

// update prompts all fake e-publications in the testServerGroup to randomly
// replace numLinks links with new ones
func (tsg *testServerGroup) update(numLinks int) {

	for i := range tsg.sites {
		// sort updates by timestamp and grab the most recent one
		listings := make([]int, 0, tsg.sites[i].numLinks)
		for k := range tsg.sites[i].updates {
			listings = append(listings, int(k))
		}
		listings = sort.IntSlice(listings)
		latest := listings[len(listings)-1]
		tsg.sites[i].updates[time.Now().Unix()] = tsg.sites[i].refreshContent(
			tsg.sites[i].updates[int64(latest)], numLinks)
	}

}

// fakeEPublication contains all the data needed to manage an HTTP endpoint
// for a fake e-publication
type fakeEPublication struct {
	id       string
	numLinks int
	server   *httptest.Server
	// Intended to map the epoch seconds timestamp of an update to the
	// mockArticleListings the update contains. Using a timestamp as a key
	// should make updates easier to sort/search.
	updates map[int64][]mockArticleListing
}

// close gracefully shuts down all servers in the TestServerGroup
func (tsg *testServerGroup) close() {
	// If there are no servers to close, do nothing
	if len(tsg.sites) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(tsg.sites))

	for i := range tsg.sites {
		go func(ts *httptest.Server, wg *sync.WaitGroup) {
			ts.Close()
			wg.Done()
		}(tsg.sites[i].server, &wg)
	}
	wg.Wait()
}

// urls returns (in string form) the URL of each server in the
// server group. Used for querying (or configuring queries for)
// the test servers.
func (tsg *testServerGroup) urls() []string {
	u := make([]string, len(tsg.sites), len(tsg.sites))
	for i := range tsg.sites {
		u[i] = tsg.sites[i].server.URL
	}
	return u
}
