package linksrc

import (
	"errors"
	"io"

	"golang.org/x/net/html"
)

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
			// The selector is ambiguous--skip this item
			return Set{}, errors.New("ambiguous link selector")
		}
		if len(ns) == 0 {
			// If the link selector has no matches, this is likely
			// true of other list items as well. Return an error
			// so we can let the user know.
			return Set{}, errors.New("no links in the list item")
		}

		if ns[0].Data != "a" {
			// The link selector doesn't match a link. This is likely
			// true of other list items, so let the user know.
			return Set{}, errors.New(
				"link selector does not match a link but rather " + ns[0].Data,
			)
		}

		// Find the href attribute of the link
		var h string // The string value of n's href attribute
		for _, a := range ns[0].Attr {
			if a.Key == "href" {
				h = a.Val
			}
		}

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

		v[i] = LinkItem{
			LinkURL: h,
			Caption: caption,
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
