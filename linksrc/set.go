package linksrc

import (
	"bytes"
	"crypto/sha256"
	"divnews/storage"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"golang.org/x/net/html"

	css "github.com/andybalholm/cascadia"
)

// extractLinkURL returns the href attribute of HTML a elements.
// It returns an error if n is not an a element.
func extractLinkURL(n html.Node) (string, error) {

	// Bail out early if the node is not a link.
	if n.Data != "a" {
		return "", fmt.Errorf(
			"trying to extract a URL from a node that's not a link. Node data: %v",
			n.Data,
		)
	}

	var h string // The string value of n's href attribute

	// Find the href attribute of the link
	for _, a := range n.Attr {
		if a.Key == "href" {
			h = a.Val
		}
	}

	return h, nil

}

// extractText returns the text node of an HTML node.
// Assumes that the text node is the first child of the HTML
// node, and returns an error if the assumption isn't met.
func extractText(n html.Node) (string, error) {
	// We're assuming that the first child node of the caption element
	// will be a text node. The text node's Data contains its content.
	// See: https://godoc.org/golang.org/x/net/html#Node
	t := n.FirstChild.Data

	// For nodes that aren't text nodes, the html package
	// seems to return an empty string
	if strings.Trim(t, "\n\t ") == "" {
		return "", errors.New("expected the parent of a text node, but could not find text")
	}

	return t, nil
}

// matchOne returns a single html.Node matching selector s.
// It returns an error if there are multiple matches.
// (This exists because there is no similar function in the
// cascadia library.)
func matchOne(s css.Selector, n *html.Node) (*html.Node, error) {

	ns := s.MatchAll(n)

	if len(ns) > 1 {
		return &html.Node{}, fmt.Errorf("ambiguous selector: returned %v matches", len(ns))
	}

	if len(ns) == 0 {
		return &html.Node{}, errors.New("found no matches")
	}

	return ns[0], nil // We know at this point that there must be a single match

}

// NewSet initializes a new collection of listed link items for an HTML
// document Reader and link source configuration.
func NewSet(r io.Reader, c Config) (Set, error) {
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

	conf, err := c.parse()

	if err != nil {
		return Set{}, err
	}

	// Get all items listing content to link to
	ls := conf.itemSelector.MatchAll(n)

	v := make([]LinkItem, len(ls))

	// Find the link URL and caption for each list item.
	for i, li := range ls {
		l, err := matchOne(conf.linkSelector, li)

		if err != nil {
			return Set{}, err
		}

		p, err := matchOne(conf.captionSelector, li)

		if err != nil {
			return Set{}, err
		}

		u, err := extractLinkURL(*l)

		if err != nil {
			return Set{}, err
		}

		t, err := extractText(*p)

		if err != nil {
			return Set{}, err
		}

		v[i] = LinkItem{
			LinkURL: u,
			Caption: t,
		}
	}

	s := Set{
		Name:  conf.name,
		Items: v,
	}

	return s, nil

}

// Set represents a set of link items
type Set struct {
	Name  string // probably the publication the links came from
	Items []LinkItem
}

// Serialize makes the Set suitable for writing to disk or comparing with
// in-memory sets.
func (s Set) Serialize() ([]byte, error) {
	// One possibility was to store only a hash of the serialized Set, allowing
	// us to check it against newly scraped Sets to see if an e-publication has
	// changed its link menu. However, we need to retain the details of the Set
	// so we can compare individual links with Sets that we have recently
	// scraped. This way, we only have to email the user links that they haven't
	// already seen.
	return json.Marshal(s)
}

// IsTheSameAs indicates whether the Set has the same values/properties as
// the serialized Set in d
func (s Set) IsTheSameAs(d []byte) (bool, error) {

	b, err := s.Serialize()

	if err != nil {
		return false, err
	}

	return bytes.Equal(b, d), nil

}

// NewSince returns a Set consisting of only the Items that are absent in
// the other Set, which is intended to be from a previous scrape
func (s Set) NewSince(other Set) (Set, error) {
	s2 := s.SortItems()
	other2 := other.SortItems()
	var results []LinkItem

	for i := range s2.Items {
		n := sort.Search(len(other2.Items), func(i int) bool {
			// We don't check captions since these might change from day to day
			// as the publication changes its headlines etc. URLs, meanwhile,
			// shouldn't change.
			return other2.Items[i].LinkURL == s2.Items[i].LinkURL
		})

		// We can't find s2.Items[i] in other2.Items
		if n == len(other2.Items) {
			results = append(results, s2.Items[i])
		}
	}

	return Set{
		Name:  s.Name,
		Items: results,
	}, nil
}

// SortItems sorts the Items in a Set for comparison, returning a new sorted Set
func (s Set) SortItems() Set {
	// Copy s.Items so we can sort it in-place
	newItems := make([]LinkItem, len(s.Items), len(s.Items))
	for i := range s.Items {
		newItems[i] = s.Items[i]
	}

	// Need stability since we'll have to sort by the link URL or caption, which
	// can easily be the same length for multiple Items
	sort.SliceStable(newItems, func(i, j int) bool {
		return len(s.Items[i].Caption) < len(s.Items[j].Caption)
	})

	return Set{
		Name:  s.Name,
		Items: newItems,
	}
}

// NewKVEntry prepares the Set to be saved in the KV database
func (s Set) NewKVEntry() (storage.KVEntry, error) {

	// We simply hash the Set's name to get the key. Collisions are most likely
	// the result of grabbing content from an online publication we've checked
	// previously, so we don't want to avoid these.
	k := sha256.New()
	k.Write([]byte(s.Name))

	b, err := s.Serialize()

	if err != nil {
		return storage.KVEntry{}, err
	}

	return storage.KVEntry{
		Key:   k.Sum(nil),
		Value: b,
	}, nil

}

// LinkItem represents data for a single link item found within a
// list of links
type LinkItem struct {
	// using a string here because we'll let the downstream context deal
	// with parsing URLs etc. This comes from a website so we can't really
	// trust it.
	LinkURL string
	Caption string
}
