package linksrc

import (
	"divnews/storage"
	"errors"
	"fmt"
	"io"
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
func NewSet(r io.Reader, conf Config) (Set, error) {
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

	v := make([]LinkItem, len(ls))

	// Find the link URL and caption for each list item.
	for i, li := range ls {
		l, err := matchOne(conf.LinkSelector, li)

		if err != nil {
			return Set{}, err
		}

		p, err := matchOne(conf.CaptionSelector, li)

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
		Name:  conf.Name,
		Items: v,
	}

	return s, nil

}

// Set represents a set of link items
type Set struct {
	Name  string // probably the publication the links came from
	Items []LinkItem
}

// NewSince returns a Set consisting of only the LinkItems that we haven't yet
// stored in the database, which we're assuming are new to the Web.
func (s Set) NewItems(db *storage.BadgerDB) Set {
	var results []LinkItem

	for _, item := range s.Items {
		// The LinkItem isn't in the database--it must be new, so use it!
		_, err := db.Read(item.Key())
		if err != nil {
			results = append(results, item)
		}
	}

	return Set{
		Name:  s.Name,
		Items: results,
	}
}
