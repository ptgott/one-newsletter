package linksrc

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/alecthomas/units"
	"github.com/andybalholm/cascadia"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const maxPageSize = 1 * units.Gibibyte

// These elements are not counted when scoring html.Nodes in possible
// captions, since they are intended to modify inline text. Other html.Nodes
// that are children of these html.Nodes, however, such as divs and images
// are counted.
// https://developer.mozilla.org/en-US/docs/Web/HTML/Element#inline_text_semantics
var inlineTags = map[string]struct{}{
	"a":      {},
	"abbr":   {},
	"b":      {},
	"bdi":    {},
	"bdo":    {},
	"br":     {},
	"cite":   {},
	"code":   {},
	"data":   {},
	"dfn":    {},
	"em":     {},
	"i":      {},
	"kbd":    {},
	"mark":   {},
	"q":      {},
	"rp":     {},
	"rt":     {},
	"ruby":   {},
	"s":      {},
	"samp":   {},
	"small":  {},
	"span":   {},
	"strong": {},
	"sub":    {},
	"sup":    {},
	"time":   {},
	"u":      {},
	"var":    {},
	"wbr":    {},
}

// used for determining if a string ends with a punctuation mark
var punctuationPattern string = `[!\.?]`
var punctuationRe *regexp.Regexp = regexp.MustCompile(punctuationPattern + " ?$")

// For catching erroneous spaces before punctuation
var spaceBeforePunctuationRe *regexp.Regexp = regexp.MustCompile(`\s+(` + punctuationPattern + ")")

var wordRe *regexp.Regexp = regexp.MustCompile(`[\w-]+`)

// distanceFromRootNode returns the number of edges between html.Node n and the
// root of the HTML document tree
func distanceFromRootNode(n *html.Node) int {
	i := 0
	c := n
	for {
		if c.DataAtom == atom.Html {
			break
		}
		c = c.Parent
		i++
	}
	return i
}

// containersAreRepeating indicates whether the container html.Nodes in n have
// the same data atom but are not identical. This is used to identify HTML tags
// that are dynamically generated for each link item in a list of link items,
// since these HTML tags will include repeating HTML around each link item.
func containersAreRepeating(n []linkContainer) (bool, error) {
	if len(n) == 0 {
		return false, errors.New("not enough link containers to make a comparison")
	}

	// Compare each Node in the Node ahead of it and break on the first
	// mismatch. If we get through the loop, by the transitive property, all
	// Nodes are equal.
	for i := 0; i < len(n)-1; i++ {
		if n[i].container == nil || n[i].link == nil {
			return false,
				errors.New("at least one Node is nil, so we can't compare it to others")
		}
		if n[i].container == n[i+1].container ||
			n[i].container.DataAtom != n[i+1].container.DataAtom {
			return false, nil
		}

	}
	return true, nil
}

// linkContainer includes an html.Node that includes the "a' tag" and the
// parent html.Node that contains the entire link item. A link item includes
// the link and any possible captions.This is used for constraining the search
// for the best caption.
type linkContainer struct {
	link      *html.Node
	container *html.Node
}

// highestRepeatingContainers finds the parent Node of n such that the Parent is
// a different Node than other parents of the same type, but with an identical
// type (i.e., data atom) and distance from the root HTML node. This is used
// to identify auto-generated HTML partials containing link items.
//
// It is possible for the Nodes in n to be their own highest repeating
// containers. This happens, for example, if all the links in a list
// are immediate children of a single container.
func highestRepeatingContainers(n []*html.Node) ([]linkContainer, error) {
	type distFromRoot struct {
		distance int
		node     *html.Node
	}

	if len(n) == 0 {
		return nil, errors.New(
			"cannot find link containers for zero nodes",
		)
	}

	// Get the distance of each Node from the root Node and keep track of the
	// shortest distance. We want to start tracking the highest repeating
	// container from a point where all Nodes are the same distance from the
	// root. This way, we know that we can keep advancing up one level of
	// parentage and eventually find a level where all Nodes are equal.
	var ld int
	ds := make([]distFromRoot, len(n), len(n))
	for i := range n {
		ds[i] = distFromRoot{
			node:     n[i],
			distance: distanceFromRootNode(n[i]),
		}
		if i == 0 || ds[i].distance < ld {
			ld = ds[i].distance
		}
	}

	lc := make([]linkContainer, len(ds), len(ds))

	// Replace each Node with its parent until all Nodes are the same
	// distance from the root. Add each Node to a slice so we can compare
	// parents.
	for i, dn := range ds {
		lc[i] = linkContainer{
			link: dn.node,
		}
		for ; dn.distance > ld; dn.distance-- {
			dn.node = dn.node.Parent
		}
		// Add the parent as a container. Since these parents are all the same
		// distance from root, we can guess that they're at least a
		// link container, if not the highest possible one.
		lc[i].container = dn.node
	}

	// Assemble a map of each link container's distance from the root to the
	// associated link containers. The higher d is, the closer each container
	// is to root. This means that we can return the []*linkContainer at the
	// key equal to len(cns)-1.
	cns := make(map[int][]linkContainer)
	d := 0
	for {
		// We're at the root
		if lc[0].container.DataAtom == atom.Html {
			break
		}
		y, err := containersAreRepeating(lc)
		if err != nil {
			return nil, err
		}
		// This level is a repeating container, so keep it in memory
		if y {
			cns[d] = make([]linkContainer, len(lc), len(lc))
			copy(cns[d], lc)
		}

		for i := range lc {
			lc[i].container = lc[i].container.Parent
		}
		d++
	}

	return cns[len(cns)-1], nil

}

// textNodeInfo includes all the data required to extract text from an
// `html.Node` tree.
type textNodeInfo struct {
	// A map where each key is the parent of a text node used to extract
	// text for the caption. The map is used to prevent counting
	// duplicate parent nodes.
	nodes map[*html.Node]struct{}
	// The text of a text node and child text nodes
	text string
	// The uppermost node we want to consider when extracting text
	container *html.Node
}

// extractTextFromNode conducts a recursive search of n, limiting the search to
// containing node e and its children. If e is nil, it sets e to n. It appends
// text node data to the string result until no more child nodes remain, and
// returns the resulting string. No-op if n is nil.
//
// Performs the following operations when extracting text from a node:
//
// - Replaces divisions between block-level elements with periods.
// - Removes block-level elements that contain fewer than m words.
func extractTextFromNode(n *html.Node, e *html.Node, c string, m int) string {
	var o *html.Node = e
	if o == nil {
		o = n
	}

	// Copy the input text to assemble the return value
	r := c

	if n == nil {
		return c
	}

	b := n
	for {
		// To gather the text from this element and its children
		bc := ""
		if b.Type == html.TextNode && len(b.Data) > 0 {

			// Replace newlines and long series of spaces with
			// single spaces.
			x := regexp.MustCompile("(\\s{2,}|\\n|\\t)")
			d := x.ReplaceAllString(b.Data, " ")

			// Remove non-displaying Unicode characters by appending
			// compliant characters to a new string.
			var txt string
			for _, e := range d {
				if (e >= ' ' && e < '\u007F') || e > '\u00A0' {
					txt += string(e)
				}
			}

			// Since we add a space to the right of each text node
			// if it's missing one, prevent double spaces by
			// removing the leftmost space.
			txt = strings.TrimLeft(txt, " \t\n")

			// Separate the content of this text node from the
			// content of the next one. If this ends up being the
			// final text node, we'll trim the space later.
			if len(d) > 0 && d[len(d)-1] != ' ' {
				txt += " "
			}
			bc += txt

		}
		// Add text from the element's children
		if b.FirstChild != nil {
			bc = extractTextFromNode(b.FirstChild, o, bc, m)
		}

		// The node is a block-level element with text.
		if _, inline := inlineTags[b.Data]; b.Type == html.ElementNode &&
			!inline &&
			strings.Trim(bc, " ") != "" {

			// The block-level element has fewer than three words,
			// so ignore it.
			if len(wordRe.FindAllString(bc, -1)) <= m {
				goto nextElement
			}

			// The text doesn't doesn't end in punctuation (but not empty
			// space), so add a period. We have already extracted text from
			// all of the element's children and their siblings, so we know
			// none of the children has provided punctuation.
			if !punctuationRe.MatchString(bc) {

				// Trim the caption segment in case we have a stray space
				// before the period.
				bc = strings.TrimRight(bc, " ") + ". "

			}
		}

		// We've processed all text for the element and its children, so
		// add the text to the accumulator string.
		r += bc

	nextElement:
		// If this is the highest node we want to consider, don't check its
		// sibling
		if b != o && b.NextSibling != nil {
			b = b.NextSibling
			continue
		}
		break
	}

	return r

}

// captionCandidate records a possible caption to use for a link as well as
// the number of nodes it took to construct that caption. The autodetection
// code uses this to determine the best caption for the link.
type captionCandidate struct {
	// The text of the caption
	text string
	// Number of nodes used to calculate the score. Intended for introspection.
	nodes int
	// nodes divided by the number of words in text
	score float32
}

// extractCaptionFromContainer finds the best caption from the children of n
// and returns it as a string. Within each HTML node, it performs the following
// operations:
//
//   - If the node is a block-level element with fewer than m words, ignores the
//     node's text.
//   - Ensures that block-level text nodes end in punctuation.
//
// After extracting text from child nodes, extractCaptionFromContainer:
//
// - Truncates the caption at 20 words.
// - Ensures that there is no space before a punctuation mark.
// - Trims whitespace on either side of the caption.
func extractCaptionFromContainer(n *html.Node, m int) (string, error) {
	if n == nil {
		return "", errors.New("cannot extract a caption from a nonexistent container")
	}

	if n.DataAtom == atom.Body || n.Data == "body" {
		return "", errors.New("cannot extract a caption from an HTML body element")
	}

	c := extractTextFromNode(n, nil, "", m)

	// Truncate at 20 words
	wi := wordRe.FindAllStringIndex(c, -1)
	if len(wi) > 20 {
		c = strings.TrimRight(c[:wi[20][0]], " ") + "..."
	}

	// Remove spaces before punctuation. We may have added these erroneously
	// while appending text nodes. We need to do this here because we don't
	// have a way to store text nodes temporarily and peek ahead.
	c = spaceBeforePunctuationRe.ReplaceAllString(c, "$1")

	// Now that we've assembled a caption string, remove any
	// leading/trailing whitespace.
	c = strings.Trim(c, " \n\t")

	return c, nil

}

type pageFormat int

const (
	formatUnknown pageFormat = iota
	formatHTML
	formatRSS
	formatAtom
)

// Used for matching the opening tag that specifies whether a document is an
// HTML document or an RSS/Atom feed. Assumes the line has been lowercased.
var openingTagPattern *regexp.Regexp = regexp.MustCompile(
	`\s*(<rss[^>]*>?|<!doctype html>|<html[^>]*>?|<feed[^>]*>?)\s*`,
)

// testFormatTag returns the pageFormat associated with a line, that is, if the
// line indicates that the page follows a particular format (HTML, RSS, or
// Atom).
func testFormatTag(line string) pageFormat {
	m := openingTagPattern.FindString(strings.ToLower(line))
	if m == "" {
		return formatUnknown
	}

	switch {
	case strings.Contains(m, "rss"):
		return formatRSS
	case strings.Contains(m, "html"):
		return formatHTML
	case strings.Contains(m, "feed"):
		return formatAtom
	}
	return formatUnknown
}

// autoDetectLinkItems uses the configured link selector to return a map of link
// URLs to LinkItems. Sends status messages and LinkItems to the provided
// channels, closing the channels when it has finished.  an email. n must be the
// root element.
func autoDetectLinkItems(r io.Reader, conf Config, links chan LinkItem, messages chan string) {
	// Copy r into two buffers. One is used for checking whether r is an HTML
	// document or RSS/Atom feed. The other is for downstream processing once we
	// have determined the kind of document we're dealing with.
	var testbuf bytes.Buffer
	var downstream bytes.Buffer
	wr := io.MultiWriter(&testbuf, &downstream)
	io.Copy(wr, io.LimitReader(r, int64(maxPageSize)))

	lines := bufio.NewScanner(&testbuf)
	lines.Split(bufio.ScanLines)
	var pf pageFormat
	for lines.Scan() {
		f := testFormatTag(lines.Text())
		if f == formatUnknown {
			continue
		}
		pf = f
		break
	}
	switch pf {
	case formatHTML:
		detectHTMLLinkItems(&downstream, conf, links, messages)
	case formatRSS, formatAtom:
		detectRSSLinkItems(&downstream, conf, links, messages)
	default:
		messages <- "could not detect a format for the page"
		close(messages)
		close(links)
	}
}

func detectHTMLLinkItems(r io.Reader, conf Config, links chan LinkItem, messages chan string) {
	n, err := html.Parse(r)
	if n == nil || err != nil {
		messages <- "Could not parse the HTML of this page."
		close(links)
		close(messages)
		return
	}

	// We're entering URL-only mode. Find all links and repeating containers
	// around those links, even if there are multiple kinds of repeating
	// containers.
	if conf.LinkSelector == nil {
		conf.LinkSelector = cascadia.MustCompile("a")
	}

	if n.Parent != nil {
		messages <- "The provided HTML node is not the root HTML node. This is a bug."
		close(links)
		close(messages)
		return
	}

	m := conf.LinkSelector.MatchAll(n)
	if len(m) == 0 {
		messages <- fmt.Sprintf(
			"The link selector you configured for %v did not match any HTML elements. ",
			conf.URL.String()) +
			"Try the request from your browser or curl and check for any issues."
		close(links)
		close(messages)
		return
	}

	// Find groups of links by:
	// - Concatenating the data atoms of each node's ancestors into a
	//   string, e.g., "adivliol", then take an MD5 hash of the string.
	// - Use that hash to identify groups of links
	// - Find the highest repeating container for each group, e.g., the HtML
	//   node that we can use to extract a caption.
	grp := make(map[[md5.Size]byte][]*html.Node)
	for _, nd := range m {
		ancestors := ""
		for c := nd; c.Parent != nil && c.Parent.DataAtom != atom.Html; c = c.Parent {
			ancestors += c.DataAtom.String()
		}
		h := md5.Sum([]byte(ancestors))
		if _, ok := grp[h]; !ok {
			grp[h] = []*html.Node{}
		}
		grp[h] = append(grp[h], nd)
	}

	for _, g := range grp {
		h, err := highestRepeatingContainers(g)

		if err != nil {
			messages <- err.Error()
		}
		for _, c := range h {
			t, err := extractCaptionFromContainer(c.container, conf.ShortElementFilter)
			if err != nil {
				messages <- err.Error()
				continue
			}
			for _, a := range c.link.Attr {
				if a.Key != "href" {
					continue
				}
				u, err := url.Parse(a.Val)

				if err != nil {
					messages <- fmt.Sprintf("Cannot parse the link URL %v", u)
					continue
				}

				links <- LinkItem{
					LinkURL: getDisplayURL(conf.URL, *u),
					Caption: t,
				}
			}
		}
	}
	close(links)
	close(messages)
}

var feedStartTag = regexp.MustCompile(`<(rss|feed)`)

// detectRSSLinkItems sends link items to the links channel and error messages
// to the messages channel. It assumes that r is a valid RSS feed.
func detectRSSLinkItems(r io.Reader, conf Config, links chan LinkItem, messages chan string) {
	f, err := gofeed.NewParser().Parse(r)
	if err != nil {
		messages <- fmt.Sprintf("cannot parse the feed: %v", err)
		close(links)
		close(messages)
		return
	}

	for _, item := range f.Items {
		var c string
		if item.Title != "" {
			c = item.Title
		} else {
			c = item.Description
		}

		links <- LinkItem{
			LinkURL: item.Link,
			Caption: c,
		}
	}
	close(links)
	close(messages)
}
