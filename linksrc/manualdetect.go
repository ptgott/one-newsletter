package linksrc

import (
	"fmt"
	"net/url"

	"golang.org/x/net/html"
)

// manuallyDetectLinkItems uses the configured link item, link, and caption
// selectors to return a map of link URLs to LinkItems. Also returns a slice of
// status messages to add to an email.
func manuallyDetectLinkItems(n *html.Node, conf Config) (map[string]LinkItem, []string) {
	s := []string{}
	v := make(map[string]LinkItem)

	if conf.ItemSelector == nil {
		s = append(s, "Could not parse the link item selector.")
		return v, s
	}

	if conf.LinkSelector == nil {
		s = append(s, "Could not parse the link selector.")
		return v, s
	}

	// Get all items listing content to link to
	ls := conf.ItemSelector.MatchAll(n)

	for i := range ls {
		ns := conf.LinkSelector.MatchAll(ls[i])
		if len(ns) > 1 {
			s = append(s, "The link selector is ambiguous, so we couldn't parse any link items.")
			return v, s
		}
		if len(ns) == 0 {
			// If the link selector has no matches, this is likely
			// true of other list items as well. Return an error
			// so we can let the user know.
			s = append(s, "There are no links in the list item. Double-check your configuration.")
			return v, s
		}

		if ns[0].Data != "a" {
			// The link selector doesn't match a link. This is likely
			// true of other list items, so let the user know.
			s = append(s, fmt.Sprintf("The link selector does not match a link but rather %v.", ns[0].Data))
			return v, s
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
			s = append(s, fmt.Sprintf("Cannot parse the link URL %v", u))
			return v, s
		}

		h = conf.URL.Scheme + "://" + conf.URL.Host + u.Path

		if conf.CaptionSelector == nil {
			s = append(s, "Could not parse the caption selector.")
			return v, s
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

		v[h] = LinkItem{
			LinkURL: h,
			Caption: caption,
		}
	}

	return v, s

}
