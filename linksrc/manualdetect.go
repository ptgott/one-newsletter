package linksrc

import (
	"fmt"
	"io"
	"net/url"

	"golang.org/x/net/html"
)

// manuallyDetectLinkItems uses the configured link item, link, and caption
// selectors. Sends LinkItems and messages to add to an email to the provided
// channels.
func manuallyDetectLinkItems(r io.Reader, conf Config, links chan LinkItem, messages chan string) {
	n, err := html.Parse(r)
	if n == nil || err != nil {
		messages <- "Could not parse the HTML of this page."
		close(links)
		close(messages)
		return
	}

	if conf.ItemSelector == nil {
		messages <- "Could not parse the link item selector."
		close(links)
		close(messages)
		return
	}

	if conf.LinkSelector == nil {
		messages <- "Could not parse the link selector."
		close(links)
		close(messages)
		return
	}

	// Get all items listing content to link to
	ls := conf.ItemSelector.MatchAll(n)

	for i := range ls {
		ns := conf.LinkSelector.MatchAll(ls[i])
		if len(ns) > 1 {
			messages <- "The link selector is ambiguous, so we couldn't parse any link items."
			close(links)
			close(messages)
			return
		}
		if len(ns) == 0 {
			// If the link selector has no matches, this is likely
			// true of other list items as well. Return an error
			// so we can let the user know.
			messages <- "There are no links in the list item. Double-check your configuration."
			close(links)
			close(messages)
			return
		}

		if ns[0].Data != "a" {
			// The link selector doesn't match a link. This is likely
			// true of other list items, so let the user know.
			messages <- fmt.Sprintf("The link selector does not match a link but rather %v.", ns[0].Data)
			close(links)
			close(messages)
			return
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
			messages <- fmt.Sprintf("Cannot parse the link URL %v", u)
			close(links)
			close(messages)
			return
		}

		if conf.CaptionSelector == nil {
			messages <- "Could not parse the caption selector."
			close(links)
			close(messages)
			return
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

		links <- LinkItem{
			LinkURL: getDisplayURL(conf.URL, *u),
			Caption: caption,
		}
	}

	close(links)
	close(messages)
	return

}
