package linksrc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

// NewSet initializes a new collection of listed link items for an HTML
// document Reader, link source configuration, and HTTP status code (which
// is treated as a 200 OK if not set)
func NewSet(ctx context.Context, r io.Reader, conf Config, code int) Set {
	s := Set{
		items: map[string]LinkItem{},
	}
	items := make(map[string]LinkItem)

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		log.Info().Msgf(
			"processed %v items for link source %q in %v ms",
			len(items),
			conf.Name,
			elapsed.Milliseconds(),
		)
	}()

	if r == nil {
		s.AddMessage("could not read the HTML document in order to parse it")
		return s
	}

	codesToMessages := map[int]string{
		403: "We don't have permission to get links from this website. Check your configuration.",
		404: "We couldn't find the website at this URL. Maybe it changed?",
		429: "We were rate limited. You should change your configuration to check this site less frequently.",
	}

	c, ok := codesToMessages[code]

	if ok {
		s.AddMessage(c)
	}

	if !ok && code == 400 {
		s.AddMessage(fmt.Sprintf("Got a %v error sending the scrape request—check your config.", code))
	}

	if !ok && code-(code%100) == 500 {
		s.AddMessage(fmt.Sprintf("Got a %v error sending the scrape request—check manually to see if this is temporary.", code))
	}

	if !ok && code >= 600 {
		s.AddMessage(fmt.Sprintf("Unexpected status code %v. Try visiting the site manually.", code))
	}

	s.Name = conf.Name

	// The rest of this function is just processing HTML, so bail early on
	// unsuccessful responses. A zero is treated as a 200, since that's the
	// default if the code is unset.
	if code-(code%100) != 200 && code != 0 {
		return s
	}

	// Note that the following quick.Check function could not find an invalid
	// input for html.Parse:
	//
	// err := quick.Check(func(doc []byte) bool {
	//     r := bytes.NewReader(doc)
	//     _, err := html.Parse(r)
	//     if err != nil {
	//         return false
	//     }
	//     return true
	// }, &quick.Config{
	//     MaxCount: 10000,
	// })
	//
	// As a result, it's safe to say that we shouldn't need to handle errors
	// for html.Parse.
	n, _ := html.Parse(r)

	if n == nil {
		s.AddMessage("Could not parse the HTML of this page.")
		return s
	}

	linkCh := make(chan LinkItem)
	msg := make(chan string)

	if conf.ItemSelector == nil || conf.CaptionSelector == nil {
		go autoDetectLinkItems(n, conf, linkCh, msg)
	} else {
		go manuallyDetectLinkItems(n, conf, linkCh, msg)
	}

	for {
		select {
		case l, ok := <-linkCh:
			if !ok {
				goto finish
			}
			items[l.LinkURL] = l
		case g, ok := <-msg:
			if !ok {
				goto finish
			}
			s.AddMessage(g)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				s.AddMessage(fmt.Sprintf("scraper %v timed out before it could extract links", s.Name))

			}
			goto finish
		}
	}
finish:

	s.items = items

	// Fix invalid data before we enforce the item limit, since removing
	// invalid items might take us under the limit.
	s = cleanSet(s)

	// If the number of list items we scraped is over the limit, we'll
	// arbitrarily exclude some list items from our search by making the
	// length of our final result slice less than the length of the initial
	// result slice.
	var limit uint

	if conf.MaxItems == 0 || len(s.items) < int(conf.MaxItems) {
		// i.e., disregard the limit if it doesn't apply
		limit = uint(len(s.items))
	} else {
		limit = conf.MaxItems
	}

	s.items = enforceLimit(s.items, limit)

	return s

}

// enforceLimit returns a copy of v after removing enough link items to satisfy
// limit.
func enforceLimit(v map[string]LinkItem, limit uint) map[string]LinkItem {
	m := make(map[string]LinkItem, limit)

	var i uint = 0
	for j := range v {
		if i < limit {
			m[j] = v[j]
		}
		i++
	}
	return m

}

// cleanSet prepares s for storage and email, returning a copy of s with
// unexpected features removed. In particular, cleanSet removes empty link items
// from the input Set.
func cleanSet(s Set) Set {
	p := Set{}
	p.Name = s.Name
	p.messages = s.messages
	p.items = make(map[string]LinkItem)

	for k, v := range s.items {
		if strings.Trim(v.Caption, "\n\t ") != "" {
			p.items[k] = v
		}

	}

	return p
}

// Set represents a set of link items. It's not meant to be modified by
// concurrent goroutines.
type Set struct {
	// The publication that the links came from
	Name string
	// LinkItems managed by the Set. Should not get and set keys directly,
	// but rather via the functions AddLinkItem, RemoveLinkItem, and LinkItems
	items map[string]LinkItem
	// Messages to include in an email, e.g., due to errors
	messages []string
}

// RemoveLinkItem removes the LinkItem from the Set. Not to be used
// concurrently
func (s *Set) RemoveLinkItem(li LinkItem) {
	delete(s.items, li.LinkURL)
}

// LinkItems returns all of the LinkItems managed by the Set
func (s *Set) LinkItems() []LinkItem {
	is := make([]LinkItem, len(s.items), len(s.items))
	var i int
	for _, v := range s.items {
		is[i] = v
		i++
	}
	return is
}

// CountLinkItems returns the number of LinkItems managed by the Set
func (s *Set) CountLinkItems() int {
	return len(s.items)
}

// AddMessage adds a message to the Set for displaying later in an email. These
// messages are used only for ad hoc notes that don't belong in a LinkItem,
// such as error messages. Messages should be complete sentences.
func (s *Set) AddMessage(msg string) {
	s.messages = append(s.messages, msg)
}

// Messages returns all of the ad-hoc messages for the Set
func (s *Set) Messages() []string {
	return s.messages
}
