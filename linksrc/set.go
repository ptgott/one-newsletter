package linksrc

import (
	"fmt"
	"io"
	"net/url"

	"golang.org/x/net/html"
)

// NewSet initializes a new collection of listed link items for an HTML
// document Reader, link source configuration, and HTTP status code (which
// is treated as a 200 OK if not set)
func NewSet(r io.Reader, conf Config, code int) Set {
	s := Set{
		items: map[string]LinkItem{},
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

	// Get all items listing content to link to
	ls := conf.ItemSelector.MatchAll(n)
	var limit uint

	if conf.MaxItems == 0 || len(ls) < int(conf.MaxItems) {
		// i.e., disregard the limit if it doesn't apply
		limit = uint(len(ls))
	} else {
		limit = conf.MaxItems
	}

	// Find the link URL and caption for each list item. Note that if the
	// number of list items we scraped is over the limit, we'll arbitrarily
	// exclude some list items from our search by making the length of our
	// final result slice less than the length of the initial result slice.
	v := make([]LinkItem, limit)
	for i := range v {
		ns := conf.LinkSelector.MatchAll(ls[i])
		if len(ns) > 1 {
			s.AddMessage("The link selector is ambiguous, so we couldn't parse any link items.")
			return s
		}
		if len(ns) == 0 {
			// If the link selector has no matches, this is likely
			// true of other list items as well. Return an error
			// so we can let the user know.
			s.AddMessage("There are no links in the list item. Double-check your configuration.")
			return s
		}

		if ns[0].Data != "a" {
			// The link selector doesn't match a link. This is likely
			// true of other list items, so let the user know.
			s.AddMessage(fmt.Sprintf("The link selector does not match a link but rather %v.", ns[0].Data))
			return s
		}

		// Find the href attribute of the link
		var h string // The string value of n's href attribute
		for _, a := range ns[0].Attr {
			if a.Key == "href" {
				h = a.Val
			}
		}

		u, err := url.Parse(h)

		if err != nil {
			s.AddMessage(fmt.Sprintf("Cannot parse the link URL %v", u))
			return s
		}

		h = conf.URL.Scheme + "://" + conf.URL.Host + u.Path

		cs := conf.CaptionSelector.MatchAll(ls[i])
		var caption string
		if len(cs) == 0 {
			// No captions in this item--skip it
			caption = ""
		}
		if len(cs) > 1 {
			// The caption is ambiguous. Keep the link, since there's
			// still value there, but let the user know.
			caption = "[Missing caption due to ambiguous selector]"
		}

		if len(cs) == 1 {
			// We're assuming that the first child node of the caption element
			// will be a text node. The text node's Data contains its content.
			// See: https://godoc.org/golang.org/x/net/html#Node
			caption = cs[0].FirstChild.Data

		}

		s.AddLinkItem(LinkItem{
			LinkURL: h,
			Caption: caption,
		})
	}

	return s

}

// Set represents a set of link items. It's not meant to be modified by
// concurrent goroutines.
type Set struct {
	Name  string              // probably the publication the links came from
	items map[string]LinkItem // LinkItems managed by the Set. Keys shouldn't be
	// get and set directly, but rather via the functions AddLinkItem,
	// RemoveLinkItem, and LinkItems
	messages []string // messages to include in an email, e.g., due to errors
}

// AddLinkItem stores the LinkItem within the Set. Not to be used concurrently.
func (s *Set) AddLinkItem(li LinkItem) {
	if s.items == nil {
		s.items = map[string]LinkItem{}
	}
	s.items[li.LinkURL] = li
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
