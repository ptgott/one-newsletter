package linksrc

import (
	"io"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// quickURL creates a url.URL from a string without
// returning an error. Use this to create URLs inline without
// error handling when we know the URL will be valid.
func quickURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func TestNewSet(t *testing.T) {
	type testCase struct {
		HTML     string
		Expected Set
		IsError  bool
	}

	testCases := []testCase{
		testCase{
			HTML: `<!doctype html5>
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
</html>`,
			IsError: false,
			Expected: Set{
				Items: []Meta{
					Meta{
						LinkURL: *quickURL("www.example.com/stories/hot-take"),
						Caption: "This is a hot take!",
					},
					Meta{
						LinkURL: *quickURL("www.example.com/stories/stuff-happened"),
						Caption: "Stuff happened today, yikes.",
					},
					Meta{
						LinkURL: *quickURL("www.example.com/storiesreally-true"),
						Caption: "Is this supposition really true?",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		// Define a linksrc.RawConfig for use in all test cases. This introduces
		// a coupling with linksrc.Validate() in this test suite, which
		// isn't great, but there's no other way to arrive at a linksrc.Config
		// without replicating the logic of linksrc.Validate().
		rc := RawConfig{
			URL:             "http://www.example.com", // Not actually used here
			ItemSelector:    "div#mostRead ul li",
			CaptionSelector: "span span.itemName",
			LinkSelector:    "a",
		}

		c, err := Validate(rc)

		if err != nil {
			t.Fatalf(
				`Could not validate the RawConfig in TestNewSet. This is an issue with the test suite, not the code--the config should always be valid: %v`,
				err,
			)
		}

		r := io.Reader(strings.NewReader(tc.HTML))
		st, err := NewSet(r, c)

		if (err != nil) != tc.IsError {
			t.Errorf(
				"Did not get the expected error status.\nCase: %v\nError: %v",
				tc,
				err,
			)
		}

		if !reflect.DeepEqual(tc.Expected, st) {
			t.Errorf(
				"Did not get the expected result.\nExpected: %v\nActual: %v",
				tc.Expected,
				st,
			)
		}
	}

}
