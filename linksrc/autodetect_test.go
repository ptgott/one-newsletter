package linksrc

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"github.com/andybalholm/cascadia"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// An unordered list of links under a div. As clean an example as we can get.
const basicLinkListDiv = `<!doctype html5>
<html>
<head>
</head>
<body>
	<h1>This is my cool website</h1>
	<div id="mostRead">
		<h2>Most read posts today</h2>
		<ul>
			<li>
				<img src="img1.png" alt="A cool image">
				<span class="itemHolder">
					<span class="itemNumber">1.</span>
					<span class="itemName">This is a hot take!</span>
				</span>
				<a href="http://www.example.com/stories/hot-take">
				Click here
				</a>
			</li>
			<li>
				<img src="img2.png" alt="This is an image">
				<span class="itemHolder">
					<span class="itemNumber">2.</span>
					<span class="itemName">Stuff happened today, yikes.</span>
				</span>
				<a href="http://www.example.com/stories/stuff-happened">
				Click here
				</a>
			</li>
			<li>
				<img src="img3.png" alt="This is also an image">
				<span class="itemHolder">
					<span class="itemNumber">3.</span>
					<span class="itemName">Is this supposition really true?</span>
				</span>
				<a href="http://www.example.com/storiesreally-true">
				Click here
				</a>
			</li>
		<ul>
	</div>
</body>
</html>`

func TestContainersAreRepeating(t *testing.T) {
	cases := []struct {
		description       string
		expected          bool
		expectedErrSubstr string // blank if no error expected
		htmlBody          string
		selector          string
	}{
		{
			description:       "straightforward case",
			expected:          true,
			expectedErrSubstr: "",
			selector:          "li a",
			htmlBody: `<html>
<body>
	<ul>
		<li><a href="/page1">This is one list item</a></li>
		<li><a href="/page2">This is a second list item</a></li>
		<li><a href="/page3">This is a third list item</a></li>
	</ul>
</body>
</html>`,
		},
		{
			description:       "different links, same container",
			expected:          false,
			expectedErrSubstr: "",
			selector:          "li",
			htmlBody: `<html>
<body>
	<ul>
		<li><a href="/page1">This is one list item</a></li>
		<li><a href="/page2">This is a second list item</a></li>
		<li><a href="/page3">This is a third list item</a></li>
	</ul>
</body>
</html>`,
		},
		{
			description:       "one matching link container",
			expected:          true,
			expectedErrSubstr: "",
			selector:          "li a",
			htmlBody: `<html>
<body>
	<ul>
		<li><a href="/page1">This is one list item</a></li>
	</ul>
</body>
</html>`,
		},
		{
			description:       "no matching link containers",
			expected:          false,
			expectedErrSubstr: "not enough link containers",
			selector:          "li a",
			htmlBody: `<html>
<body>
	<ul>
	</ul>
</body>
</html>`,
		},
		{
			description:       "one container with multiple links",
			expected:          false,
			expectedErrSubstr: "",
			selector:          "a",
			htmlBody: `<html>
<head></head>
<body>
	<div id="links">
	<p>Something happened the other day.</p>
	<a href="/page1">Read more</a>
	<p>Something else happened the other day.</p>
	<a href="/page2">Read more</a>
	<p>Here is an opinion.</p>
	<a href="/page3">Read more</a>
	</div>
</body>
</html>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			r := strings.NewReader(tc.htmlBody)
			n, err := html.Parse(r)

			// This is an error with the test setup, not
			// containersAreRepeating. Should only happen while writing tests.
			if err != nil {
				t.Fatal(err)
			}
			ns := cascadia.MustCompile(tc.selector).MatchAll(n)

			l := make([]linkContainer, len(ns), len(ns))

			for i, d := range ns {
				l[i] = linkContainer{
					// In this test case, this doesn't have to be an a element.
					link:      d,
					container: d.Parent,
				}
			}

			b, err := containersAreRepeating(l)

			if err == nil && tc.expectedErrSubstr != "" {
				t.Fatal("expected an error but got none")
			}

			if err != nil {
				if tc.expectedErrSubstr == "" {
					t.Fatalf("expected no error but got %v", err)
				}

				if !strings.Contains(err.Error(), tc.expectedErrSubstr) {
					t.Fatalf("expected error containing %v but got %v", tc.expectedErrSubstr, err)
				}
			}

			if b != tc.expected {
				t.Fatalf("expected return value of %v but got %v", tc.expected, b)
			}

		})
	}

}

func TestBuildHTMLTree(t *testing.T) {
	var expected = `<body>
<div data[level]="0">
<p data[level]="1">word word word</p>
<p data[level]="1">word word word</p>
</div>
<div data[level]="0">
<p data[level]="1">word word word</p>
<p data[level]="1">word word word</p>
</div>
</body>`
	n := html.Node{
		Data:     "body",
		DataAtom: atom.Body,
		Type:     html.ElementNode,
	}

	buildHTMLTree(&n, 2, 0, 2, 3, 0)
	var buf bytes.Buffer
	if err := html.Render(&buf, &n); err != nil {
		t.Fatal(err)
	}

	s := buf.String()
	s = strings.ReplaceAll(s, "><", ">\n<")

	assert.Equal(t, expected, s)

}

// buildHTMLTree recursively adds children to parent up to maxLevels. It adds
// childrenPerNode children to each interior node, and inserts a text node of
// wordsPerTextNode words to each leaf node. addedNodes keeps track of how
// many nodes have been inserted into the tree so far. The function returns the
// number of nodes in the tree.
func buildHTMLTree(
	parent *html.Node,
	maxLevels int,
	current int,
	childrenPerNode int,
	wordsPerTextNode int,
	addedNodes int,
) int {
	// Count the parent node
	if addedNodes == 0 {
		addedNodes = 1
	}

	if maxLevels <= 1 {
		return addedNodes
	}

	if maxLevels == current {
		return addedNodes
	}

	var c html.Node
	if current == maxLevels-1 {
		c = html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.P,
			Data:     "p",
		}

		w := make([]string, wordsPerTextNode, wordsPerTextNode)
		for i := range w {
			w[i] = "word"
		}
		c.AppendChild(&html.Node{
			Type:     html.TextNode,
			DataAtom: atom.Plaintext,
			Data:     strings.Join(w, " "),
		})
		addedNodes++
	} else {
		c = html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Div,
			Data:     "div",
		}
	}

	for i := 0; i < childrenPerNode; i++ {
		k := c

		k.Attr = append(k.Attr, html.Attribute{
			Namespace: "",
			Key:       "data[level]",
			Val:       strconv.Itoa(current),
		})
		parent.AppendChild(&k)
	}
	current++

	for n := parent.FirstChild; n != nil; n = n.NextSibling {
		addedNodes = buildHTMLTree(
			n,
			maxLevels,
			current,
			childrenPerNode,
			wordsPerTextNode,
			addedNodes,
		)
	}

	return addedNodes

}

func BenchmarkExtractCaptionFromContainer(b *testing.B) {
	cases := []struct {
		description      string
		levels           int
		childrenPerNode  int
		wordsPerTextNode int
	}{
		{
			description:      "1,025 nodes, 2 children per node, 5-word text nodes",
			childrenPerNode:  2,
			levels:           11,
			wordsPerTextNode: 5,
		},
		{
			description:      "1,025 nodes, 2 children per node, 1000-word text nodes",
			childrenPerNode:  2,
			levels:           11,
			wordsPerTextNode: 1000,
		},
		{
			description:      "17 nodes, 2 children per node, 10000-word text nodes",
			childrenPerNode:  2,
			levels:           5,
			wordsPerTextNode: 10000,
		},
	}

	for _, c := range cases {
		b.Run(c.description, func(b *testing.B) {
			n := html.Node{
				DataAtom: atom.Body,
				Data:     "body"}
			addedNodes := buildHTMLTree(
				&n,
				c.levels,
				0,
				c.childrenPerNode,
				c.wordsPerTextNode,
				0,
			)
			fmt.Println("built an HTML tree with", addedNodes, "nodes")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := extractCaptionFromContainer(&n, 3)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// appendLeafNodes appends a child to parent until it reaches the
// max number of edges between root and leaf. It returns the final leaf
// Node
func appendLeafNodes(parent *html.Node, max int, current int) *html.Node {
	if max == current {
		return parent
	}

	c := html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Div,
	}

	parent.AppendChild(&c)
	current++
	return appendLeafNodes(&c, max, current)
}

func TestDistanceFromRootNode(t *testing.T) {

	// l is the number of edges to expect between a Node and the HTML root
	// node
	if err := quick.Check(func(l uint8) bool {

		n := html.Node{
			Type:     html.ElementNode,
			DataAtom: atom.Html,
		}

		c := appendLeafNodes(&n, int(l), 0)
		if d := distanceFromRootNode(c); d != int(l) {
			fmt.Printf("got distance %v from input %v\n", d, l)
			return false
		}
		return true
	}, &quick.Config{
		MaxCount: 1000,
	}); err != nil {
		qce := err.(*quick.CheckError)
		t.Errorf("failed after %v iterations with input %v",
			qce.Count,
			qce.In,
		)
	}
}

func TestHighestRepeatingContainers(t *testing.T) {
	cases := []struct {
		description                string
		expectedlinkContainerCount int
		expectedContainerAtom      atom.Atom
		linkSelector               string
		body                       io.Reader
		expectError                bool
	}{
		{
			description:                "straightforward case",
			expectedlinkContainerCount: 3,
			expectedContainerAtom:      atom.Li,
			linkSelector:               "a",
			body:                       strings.NewReader(basicLinkListDiv),
			expectError:                false,
		},
		{
			description:                "longer html doc with doctype",
			expectedlinkContainerCount: 3,
			expectedContainerAtom:      atom.Li,
			linkSelector:               "a",
			body:                       mustReadFile(path.Join("testdata", "straightforward.html"), t),
			expectError:                false,
		},
		{
			description:  "the link is the highest repeating container",
			linkSelector: "a",
			body: strings.NewReader(`<html>
<head></head>
<body>
	<div id="links">
	<p>Something happened the other day.</p>
	<a href="/page1">Read more</a>
	<p>Something else happened the other day.</p>
	<a href="/page2">Read more</a>
	<p>Here is an opinion.</p>
	<a href="/page3">Read more</a>
	</div>
</body>
</html>`),
			expectError:                false,
			expectedlinkContainerCount: 3,
			expectedContainerAtom:      atom.A,
		},
		{
			description:  "repeating container with an ad interruption",
			linkSelector: "article a",
			body: strings.NewReader(`<html>
<head></head>
<body>
	<div id="links">
		<article>
			<p>Something happened the other day.</p>
			<a href="/page1">Read more</a>
		</article>
		<article>
			<p>Something else happened.</p>
			<a href="/page2">Read more</a>
		</article>
		<div>
			<p>Save on your car insurance!</p>
			<a href="example.com/ad"><img src="example.com/ad.jpg"></img></a>
		</div>
		<article>
			<p>Another thing happend.</p>
			<a href="/page3">Read more</a>
		</article>
	</div>
</body>
</html>`),
			expectError:                false,
			expectedlinkContainerCount: 3,
			expectedContainerAtom:      atom.Article,
		},
		{
			description:  "multiple links in container",
			linkSelector: "body > article > a",
			body: strings.NewReader(`
<!doctype html>
<html>
<head></head>
<body>
	<article>
		<a href="example.com/1">This is a link</a>
		<div>
			<article>
				<a href="example.com/about">About</a>
			</article>
		</div>
	</article>
	<article>
		<a href="example.com/2">This is a link</a>
		<div>
			<article>
				<a href="example.com/about">About</a>
			</article>
		</div>
	</article>
	<article>
		<a href="example.com/3">This is a link</a>
		<div>
			<article>
				<a href="example.com/about">About</a>
			</article>
		</div>
	</article>
</body>
</html>`),
			expectError:                false,
			expectedlinkContainerCount: 3,
			expectedContainerAtom:      atom.Article,
		},
		{
			description:                "The Baffler",
			linkSelector:               "div > article > div > a",
			body:                       mustReadFile(path.Join("testdata", "baffler-many-links.html"), t),
			expectError:                false,
			expectedlinkContainerCount: 12,
			expectedContainerAtom:      atom.Article,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			n, err := html.Parse(tc.body)
			if err != nil {
				t.Error(err)
			}
			ns := cascadia.MustCompile(tc.linkSelector).MatchAll(n)
			rcs, err := highestRepeatingContainers(ns)
			if (err != nil) != tc.expectError {
				t.Errorf("expected error status of %v but got %v with error %v", tc.expectError, err != nil, err)
			}
			if len(rcs) != tc.expectedlinkContainerCount {
				t.Fatalf("expected %v link containers but got %v", tc.expectedlinkContainerCount, len(rcs))
			}
			for _, c := range rcs {
				if c.container.DataAtom != tc.expectedContainerAtom {
					t.Errorf("expected data atom %v but got %v", tc.expectedContainerAtom, c.container.DataAtom)
				}
			}

		})
	}
}

func TestExtractCaptionFromContainer(t *testing.T) {
	cases := []struct {
		description      string
		html             string
		expected         string
		expectErr        bool
		selector         string
		minTextNodeWords int
	}{
		{
			description: "straightforward case",
			selector:    "div",
			html: `<html>
<head></head>
<body>
<div>
    <p>This is the beginning of a long, multi-tag <a href="#">text node</a>. </p>
    <p>This is the end.</p>
</div>
</body>	
</html>`,
			expected: "This is the beginning of a long, multi-tag text" +
				" node. This is the end.",
			expectErr: false,
		},
		{
			description:      "text nodes in block elements unrelated to a caption",
			selector:         "li",
			minTextNodeWords: 3,
			html: `<li>
	<img src="img1.png" alt="A cool image">
	<div class="itemHolder">
		<div class="itemNumber">1. </div>
		<div class="itemName">This is a hot take!</div>
	</div>
	<a href="http://www.example.com/stories/hot-take">
	Click here
	</a>
</li>`,
			expected:  "This is a hot take! Click here.",
			expectErr: false,
		},
		// Based on actual HTML from "aldaily.com". Original link replaced
		// with an example.com link.
		{
			description: "Arts and Letters Daily link paragraph",
			selector:    "p",
			html: `<p>
<strong>Long novels</strong> offer pleasures that come from having traveled with a character over time. Can gimmicks reproduce that in shorter books?&nbsp;...&nbsp;<a href="https://www.example.com/magazine/archive/2022/04/jennifer-egan-goon-squad-candy-house/622831/">more&nbsp;»</a>
</p>`,
			expected: "Long novels offer pleasures that come " +
				"from having traveled with a character over time. Can " +
				"gimmicks reproduce that in shorter...",
			expectErr: false,
		},
		{
			description:      "Intelligencer feed item",
			selector:         "a",
			minTextNodeWords: 3,
			html: `<a
href="http://example.com/intelligencer/2022/04/letitia-james-is-going-after-gas-price-gougers.html"
class="feed-item article"
>
<div class="feed-item-timestamp-container">
	<div class="rubric">tish james</div>
</div>
<div class="feed-item-content small">
	<div class="content">
	<div class="rubric">tish james</div>
	<div class="headline">
		Letitia James Is Going After Gas-Price Gougers
	</div>
	<div class="byline">
		<span>By</span> <span>Kevin T. Dugan</span>
	</div>
	<div class="teaser">
		The state attorney general is probing oil-industry practices,
		as companies like Exxon rake in big bucks while consumers pay
		more.
	</div>
	</div>
</div>
</a>`,
			expected:  "Letitia James Is Going After Gas-Price Gougers. By Kevin T. Dugan. The state attorney general is probing oil-industry practices, as...",
			expectErr: false,
		},
		{
			description:      "Slate most-read item",
			selector:         "a",
			minTextNodeWords: 3,
			html: `<section class="most-engaged-teaser" data-tb-region-item="" data-tb-owning-region-name="Most Engaged" data-tb-owning-region-index="0" uniqueid="ID826716621843200544" data-tb-shadow-region-item="0-0">
<a href="https://slate.com/news-and-politics/2022/04/history-textbook-controversy-new-orleans-louisiana.html">

	<div class="most-engaged-teaser__image">
		<div class="lazyload-container"><img class="lazyautosizes lazyloaded"></div>
	</div>

	<h3 class="most-engaged-teaser__headline" data-tb-shadow-region-title="0">
		
		New Orleans’ Self-Mythology Dates Back to a Shockingly Racist Old Textbook
	</h3>

	<p class="most-engaged-teaser__byline">
Jordan Hirsch
			</p>

</a>
</section>`,
			expected:  "New Orleans’ Self-Mythology Dates Back to a Shockingly Racist Old Textbook.",
			expectErr: false,
		},
		{
			description: "block elements on the same line",
			html: `<li>
	<div>May 6:</div><div>This is something that happened <a href="http://www.example.com/stories/hot-take">today</a></div>
</li>`,
			selector:         "li",
			minTextNodeWords: 3,
			expectErr:        false,
			expected:         "This is something that happened today.",
		},
		{
			description:      "short block element",
			selector:         "div",
			minTextNodeWords: 3,
			html: `<html>
<head></head>
<body>
<div>
    <div class="byline">
      By
      <span>First</span><span>Last</span>
    </div>
    <p>This is the beginning of a long, multi-tag <a href="#">text node</a>. </p>
    <p>This is the end.</p>
</div>
</body>	
</html>`,
			expected: "This is the beginning of a long, multi-tag text" +
				" node. This is the end.",
			expectErr: false,
		},
		{
			description: "extracting from body in straightforward case",
			selector:    "body",
			html:        stringFromFile(path.Join("testdata", "straightforward.html"), t),
			expected:    "",
			expectErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			r := strings.NewReader(tc.html)
			h, err := html.Parse(r)
			if err != nil {
				t.Error(err)
			}
			s := cascadia.MustCompile(tc.selector)
			n := s.MatchFirst(h)
			c, err := extractCaptionFromContainer(n, tc.minTextNodeWords)

			if (err != nil) != tc.expectErr {
				t.Fatalf("expected error status of %v but got %v with err %v", tc.expectErr, err != nil, err)
			}

			if c != tc.expected {
				t.Fatalf("expected caption %q but got %q", tc.expected, c)
			}

		})
	}

}

func TestTestFormatTag(t *testing.T) {
	cases := []struct {
		description string
		input       string
		expected    pageFormat
	}{
		{
			description: "RSS",
			input:       `<rss version="0.91">`,
			expected:    formatRSS,
		},
		{
			description: "Atom",
			input:       `<feed xmlns="http://www.w3.org/2005/Atom">`,
			expected:    formatAtom,
		},
		{
			description: "HTML",
			input:       `<!DOCTYPE html>`,
			expected:    formatHTML,
		},
		{
			description: "HTML lowercased",
			input:       `<!doctype html>`,
			expected:    formatHTML,
		},
		{
			description: "HTML mixed case",
			input:       `<!dOcTyPe html>`,
			expected:    formatHTML,
		},
		{
			description: "HTML in quirks mode",
			input:       `<html lang="en" op="news">`,
			expected:    formatHTML,
		},
		{
			description: "HTML on same line as another tag",
			input:       "<html><head>",
			expected:    formatHTML,
		},
		{
			description: "Relevant tag after another",
			input:       `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"`,
			expected:    formatRSS,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actual := testFormatTag(c.input)
			assert.Equal(t, c.expected, actual)
		})
	}
}
