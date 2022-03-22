package linksrc

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

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

// textNodeScoreInfo includes all the data required to score a text node that
// makes up part of a larger caption candidate.
type textNodeScoreInfo struct {
	// A map where each key is the parent of a text node used to extract
	// text for the caption. The length of this map is used to calculate
	// the caption candidate's score. A map is used to prevent counting
	// duplicate parent nodes.
	nodes map[*html.Node]struct{}
	// The text of a text node and child text nodes
	text string
	// The uppermost node we want to consider when extracting text
	container *html.Node
}

// extractTextFromNode conducts a recursive depth-first search of n. It appends
// text nodes to the textNodeScoreInfo c until no more child nodes remain.
// If c is nil, begins with an empty textNodeScoreInfo.
// No-op if n is nil. Returns the final slice of caption fragments.
func extractTextFromNode(n *html.Node, c *textNodeScoreInfo) *textNodeScoreInfo {
	if n == nil {
		return c
	}

	if c == nil {
		c = &textNodeScoreInfo{
			nodes:     map[*html.Node]struct{}{},
			text:      "",
			container: n,
		}
	}

	b := n
	for {
		if b.Type == html.TextNode {
			c.text += b.Data
			// Don't count text nodes that are children of inline tags toward the
			// node count. These text nodes should be treated as part of a wider
			// passage of text.
			_, ok1 := inlineTags[b.Parent.Data]
			// Make sure we haven't counted this parent before
			_, ok2 := c.nodes[b.Parent]
			if !ok1 && !ok2 {
				c.nodes[b.Parent] = struct{}{}
			}
		}
		if b.FirstChild != nil {
			c = extractTextFromNode(b.FirstChild, c)
		}
		// If this is the highest node we want to consider, don't check its
		// sibling
		if b != c.container && b.NextSibling != nil {
			b = b.NextSibling
			continue
		}
		break
	}

	return c

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

// extractCaptionCandidate returns the captionCandidate for a given Node, i.e.,
// the text extracted from all of the text nodes within the Node, as well as
// the number of nodes required to perform the extraction.
func extractCaptionCandidate(n *html.Node) captionCandidate {
	c := extractTextFromNode(n, nil)

	w := regexp.MustCompile("\\b{2}")
	x := regexp.MustCompile("\\s{2,}|\\n")
	c.text = strings.Trim(x.ReplaceAllString(c.text, " "), " ")

	var txt string

	// Remove non-displaying Unicode characters
	for _, e := range c.text {
		if (e >= ' ' && e < '\u007F') || e > '\u00A0' {
			txt += string(e)
		}
	}

	var cc captionCandidate
	cc.text = txt
	var r int

	// Avoid dividing by zero when we calculate the score
	if len(c.nodes) == 0 {
		r = 1
	} else {
		r = len(c.nodes)
	}

	cc.nodes = r
	cc.score = float32(len(w.FindAllString(cc.text, -1))) / float32(r)

	return cc

}

// findBestCaptionFromFirstLevelChildren traverses the first-level children of
// Node n, extracts a possible caption from each, and compares these captions to
// the current best caption. It runs recursively and depth first, and returns the
// best caption candidate it finds.
func findBestCaptionFromFirstLevelChildren(n *html.Node, current captionCandidate) captionCandidate {
	var best captionCandidate = current
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		p := extractCaptionCandidate(c)
		if p.score > best.score {
			best = p
		}
		best = findBestCaptionFromFirstLevelChildren(c, best)
	}
	return best
}

// extractCaptionFromContainer finds the best caption from the children of n
// and returns it as a string.
func extractCaptionFromContainer(n *html.Node) (string, error) {
	if n == nil {
		return "", errors.New("cannot extract a caption from a nonexistent container")
	}

	var best captionCandidate = extractCaptionCandidate(n)

	best = findBestCaptionFromFirstLevelChildren(n, best)

	return best.text, nil

}

// autoDetectLinkItems uses the configured link selector to return a map of
// link URLs to LinkItems. Also returns a slice of status messages to add to
// an email. n must be the root element.
func autoDetectLinkItems(n *html.Node, conf Config) (map[string]LinkItem, []string) {
	s := []string{}
	v := make(map[string]LinkItem)

	if conf.LinkSelector == nil {
		s = append(s, "Could not parse the link selector.")
		return v, s
	}

	if n.Parent != nil {
		s = append(s, "The provided HTML node is not the root HTML node. This is a bug.")
		return v, s
	}

	m := conf.LinkSelector.MatchAll(n)
	if len(m) == 0 {
		s = append(s,
			fmt.Sprintf(
				"The link selector you configured for %v did not match any HTML elements. ",
				conf.URL.String())+
				"Try the request from your browser or curl and check for any issues.",
		)
		return v, s
	}

	h, err := highestRepeatingContainers(m)

	if err != nil {
		s = append(s, err.Error())
		return v, s
	}

	for _, c := range h {

		t, err := extractCaptionFromContainer(c.container)
		if err != nil {
			s = append(s, err.Error())
			continue
		}
		for _, a := range c.link.Attr {
			if a.Key == "href" {
				u, err := url.Parse(a.Val)

				if err != nil {
					s = append(s, fmt.Sprintf("Cannot parse the link URL %v", u))
					continue
				}

				h := conf.URL.Scheme + "://" + conf.URL.Host + u.Path
				v[h] = LinkItem{
					LinkURL: h,
					Caption: t,
				}
			}
		}
	}

	return v, s

}
