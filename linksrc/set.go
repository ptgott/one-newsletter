package linksrc

import (
	"fmt"
	"io"
	"net/url"
	"regexp"

	"golang.org/x/net/html"
)

// extractLinkURL returns the href attribute of HTML a elements.
// It returns an error if n is not an a element.
func extractLinkURL(n html.Node) (url.URL, error) {

	// Bail out early if the node is not a link.
	if n.Data != "a" {
		return url.URL{}, fmt.Errorf(
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

	u, err := url.Parse(h)

	if err != nil {
		return url.URL{}, fmt.Errorf(
			"could not extract a URL from node with data %v: %v",
			n.Data,
			err,
		)
	}

	return *u, nil

}

// extractText returns the text node of an HTML node.
// Assumes that the text node is the first child of the HTML
// node, and returns an error if the assumption isn't met.
func extractText(n html.Node) (string, error) {
	// We need to flag html.Node.Data attributes that are actually
	// HTML tag names, since html.Node.Data is a tag name for HTML nodes,
	// and the text itself for text nodes.
	r, _ := regexp.Compile("^[a-z]+$") // Not expecting this to return an error.

	// We're assuming that the first child node of the caption element
	// will be a text node. The text node's Data contains its content.
	// See: https://godoc.org/golang.org/x/net/html#Node
	t := n.FirstChild.Data

	// Unfortunately, there's no good way to distinguish the Data of an
	// HTML element (a tag name like "div" or "span") from the Data of a
	// legitimate text node, not least because you can give HTML tags
	// arbitrary names. What we can do is return an error if the caption is a
	// single, lowercased word, which is both likely to be an HTML tag and
	// a really unhelpful caption in general.
	if r.Match([]byte(t)) {
		return "", fmt.Errorf(
			"expecting a text node, but got a node with Data %v",
			n.Data,
		)
	}

	return t, nil
}

// NewSet initializes a new collection of listed link items for an HTML
// document Reader and link source configuration.
func NewSet(r io.Reader, c Config) (Set, error) {
	n, err := html.Parse(r)

	if err != nil {
		return Set{}, err
	}

	// Get all items listing content to link to
	ls := c.ItemSelector.MatchAll(n)

	v := make([]Meta, len(ls))

	// Find the link URL and caption for each list item.
	for i, li := range ls {
		l := c.LinkSelector.MatchFirst(li)
		p := c.CaptionSelector.MatchFirst(li)

		u, err := extractLinkURL(*l)

		if err != nil {
			return Set{}, err
		}

		t, err := extractText(*p)

		if err != nil {
			return Set{}, err
		}

		v[i] = Meta{
			LinkURL: u,
			Caption: t,
		}
	}

	s := Set{
		Items: v,
	}

	return s, nil

}

// Set represents a set of link items
type Set struct {
	Items []Meta
}

// Meta represents data for a single link item
type Meta struct {
	LinkURL url.URL
	Caption string
}
