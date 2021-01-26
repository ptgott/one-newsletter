package linksrc

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

// Using one HTML string in all unit tests. Just like
// in a real case, we can't change the HTML we want to
// scrape.
const testHTML = `<!doctype html5>
<html>
<head>
</head>
<body>
	<h1>This is my cool website</h1>
	<div id="mostRead">
		<h2>Most read posts today</h2>
		<ul>
			<li>
				<img src="img1.png">A cool image</img>
				<span class="itemHolder">
					<span class="itemNumber">1.</span>
					<span class="itemName">This is a hot take!</span>
				</span>
				<a href="www.example.com/stories/hot-take">
				Click here
				</a>
			</li>
			<li>
				<img src="img2.png">This is an image</img>
				<span class="itemHolder">
					<span class="itemNumber">2.</span>
					<span class="itemName">Stuff happened today, yikes.</span>
				</span>
				<a href="www.example.com/stories/stuff-happened">
				Click here
				</a>
			</li>
			<li>
				<img src="img3.png">This is also an image</img>
				<span class="itemHolder">
					<span class="itemNumber">3.</span>
					<span class="itemName">Is this supposition really true?</span>
				</span>
				<a href="www.example.com/storiesreally-true">
				Click here
				</a>
			</li>
		<ul>
	</div>
</body>
</html>`

func TestNewSet(t *testing.T) {
	type testCase struct {
		config      Config
		expected    Set
		isError     bool
		description string
	}

	testCases := []testCase{
		{
			description: "canonical/valid case",
			config: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com", // Not actually used here
				ItemSelector:    "div#mostRead ul li",
				CaptionSelector: "span span.itemName",
				LinkSelector:    "a",
			},
			isError: false,
			expected: Set{
				Name: "Example Site",
				Items: []LinkItem{
					{
						LinkURL: "www.example.com/stories/hot-take",
						Caption: "This is a hot take!",
					},
					{
						LinkURL: "www.example.com/stories/stuff-happened",
						Caption: "Stuff happened today, yikes.",
					},
					{
						LinkURL: "www.example.com/storiesreally-true",
						Caption: "Is this supposition really true?",
					},
				},
			},
		},
		{
			description: "the LinkSelector doesn't match any elements",
			config: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com", // not used here,
				ItemSelector:    "div#mostRead ul li",
				CaptionSelector: "span span.itemName",
				LinkSelector:    "p.blah",
			},
			isError:  true,
			expected: Set{},
		},
		{
			description: "the LinkSelector matches an element but not a link",
			config: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com", // not used here,
				ItemSelector:    "div#mostRead ul li",
				CaptionSelector: "span span.itemName",
				LinkSelector:    "img",
			},
			isError:  true,
			expected: Set{},
		},
		{
			description: "the CaptionSelector could match multiple elements",
			config: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com", // Not actually used here
				ItemSelector:    "div#mostRead ul li",
				CaptionSelector: "span",
				LinkSelector:    "a",
			},
			isError:  true,
			expected: Set{},
		},
		{
			description: "the CaptionSelector doesn't match the parent of a text node",
			config: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com", // Not actually used here
				ItemSelector:    "div#mostRead ul li",
				CaptionSelector: "span.itemHolder",
				LinkSelector:    "a",
			},
			isError:  true,
			expected: Set{},
		},
		{
			description: "the CaptionSelector ambiguously matches the parent of a text node",
			config: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com", // Not actually used here
				ItemSelector:    "div#mostRead ul li",
				CaptionSelector: "span span", // could match span.itemNumber or span.itemName
				LinkSelector:    "a",
			},
			isError:  true,
			expected: Set{},
		},
		{
			description: "the CaptionSelector doesn't match any elements",
			config: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com", // Not actually used here
				ItemSelector:    "div#mostRead ul li",
				CaptionSelector: "p.caption",
				LinkSelector:    "a",
			},
			isError:  true,
			expected: Set{},
		},
		{
			description: "the Config is invalid",
			config: Config{
				Name:            "Example Site",
				URL:             "blargh",
				ItemSelector:    "???",
				CaptionSelector: "!!!",
				LinkSelector:    "<>",
			},
			isError:  true,
			expected: Set{},
		},
	}

	for _, tc := range testCases {
		r := io.Reader(strings.NewReader(testHTML))
		st, err := NewSet(r, tc.config)

		if (err != nil) != tc.isError {
			t.Errorf(
				"Did not get the expected error status.\nCase: %v\nExpected: %v\nActual: %v",
				tc.description,
				tc.isError,
				err,
			)
		}

		if !reflect.DeepEqual(tc.expected, st) {
			t.Errorf(
				"Did not get the expected result.\nCase: %v\nExpected: %v\nActual: %v",
				tc.description,
				tc.expected,
				st,
			)
		}
	}

}
