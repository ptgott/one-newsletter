package linksrc

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andybalholm/cascadia"
	"gopkg.in/yaml.v2"
)

func TestUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		description   string
		shouldBeError bool
		input         string
	}{
		{
			description:   "valid case",
			shouldBeError: false,
			input: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "a"
maxItems: 5
`,
		},
		{
			description:   "negative max items",
			shouldBeError: true,
			input: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "a"
maxItems: -5
`,
		},
		{
			description:   "non-integer max items",
			shouldBeError: true,
			input: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "a"
maxItems: 2.8
`,
		},
		{
			description:   "not an object",
			shouldBeError: true,
			input:         `[]`,
		},
		{
			description:   "no url",
			shouldBeError: true,
			input: `name: site-38911
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "a"`,
		},
		{
			description:   "blank url",
			shouldBeError: true,
			input: `name: site-38911
url: ""
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "a"`,
		},
		{
			description:   "unparseable item selector",
			shouldBeError: true,
			input: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "123"
captionSelector: "p"
linkSelector: "a"`,
		},
		{
			description:   "unparseable caption selector",
			shouldBeError: true,
			input: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "ul li"
captionSelector: "123"
linkSelector: "a"`,
		},
		{
			description:   "unparseable link selector",
			shouldBeError: true,
			input: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "123"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dec := yaml.NewDecoder(bytes.NewBuffer([]byte(tc.input)))
			var c Config
			if err := dec.Decode(&c); (err != nil) != tc.shouldBeError {
				t.Errorf(
					"expected error status of %v but got %v with error %v",
					tc.shouldBeError,
					err != nil,
					err,
				)
			}
		})
	}
}

func TestUnmarshalYAMLWithMinElementWords(t *testing.T) {

	testCases := []struct {
		description                string
		config                     string
		expectedShortElementFilter int
		expectErr                  bool
	}{

		{
			description: "blank minElementWords",
			config: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "a"
maxItems: 5
`,
			expectedShortElementFilter: 3,
			expectErr:                  false,
		},
		{
			description: "minElementWords of zero",
			config: `name: site-38911
url: http://127.0.0.1:38911
itemSelector: "ul li"
captionSelector: "p"
linkSelector: "a"
maxItems: 5
minElementWords: 0
`,
			expectedShortElementFilter: 0,
			expectErr:                  false,
		},
		{
			description: "minElementWords of three, URL-only mode",
			config: `name: site-38911
url: http://127.0.0.1:38911
minElementWords: 3
`,
			expectedShortElementFilter: 3,
			expectErr:                  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			dec := yaml.NewDecoder(bytes.NewBuffer([]byte(tc.config)))
			var c Config
			if err := dec.Decode(&c); (err != nil) != tc.expectErr {
				t.Errorf(
					"expected error status of %v but got %v with error %v",
					tc.expectErr,
					err != nil,
					err,
				)
			}
			if tc.expectedShortElementFilter != c.ShortElementFilter {
				t.Errorf(
					"expected short element filter of %v but got %v",
					tc.expectedShortElementFilter,
					c.ShortElementFilter,
				)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {

	cases := []struct {
		description   string
		value         string
		shouldBeValid bool
	}{
		{
			description:   "no scheme",
			value:         "www.example.com",
			shouldBeValid: false, // Should include a scheme
		},
		{
			description:   "valid case",
			value:         "http://www.example.com/path",
			shouldBeValid: true,
		},
		{
			description:   "only hostname",
			value:         "localhost",
			shouldBeValid: false,
		},
		{
			description:   "hostname and port",
			value:         "localhost:3000",
			shouldBeValid: true,
		},
		{
			description: "no tld",
			value:       "http://www.example",
			// Technically, TLDs can resolve to IP addresses, though this is super rare.
			// See:
			// https://serverfault.com/questions/90737/how-the-heck-is-http-to-a-valid-domain-name
			shouldBeValid: true,
		},
		{
			description: "relative URL path",
			// The origin will be our own webserver, so no relative URLs
			value:         "/path",
			shouldBeValid: false,
		},
		{
			description:   "blank URL",
			value:         "",
			shouldBeValid: false,
		},
		{
			description: "just a scheme and a character",
			// Just a hash with a scheme (definitely invalid but could
			// seem valid to logic that just checks for schemes)
			value:         "http://#",
			shouldBeValid: false,
		},
		{
			description:   "includes a space",
			value:         "http://www.example test.com",
			shouldBeValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := parseURL(tc.value)

			if v := err == nil; v != tc.shouldBeValid {
				t.Errorf("Unexpected error status for %v\nWanted: %v\nGot: %v\nError: %v", tc.value, tc.shouldBeValid, v, err)
			}
		})

	}
}

func TestValidateCSSSelector(t *testing.T) {

	cases := []struct {
		description   string
		value         string
		shouldBeValid bool
	}{
		{
			description:   "nth of type",
			value:         "div ul li:nth-of-type(3)",
			shouldBeValid: true,
		},
		{
			description:   "mispelled selectors with valid character classes",
			value:         "duv il la:uth-of-type(3)",
			shouldBeValid: false,
		},
		{
			description:   "integer string",
			value:         "123",
			shouldBeValid: false,
		},
		{
			description:   "HTML element with an arbitrary tag name",
			value:         "blah",
			shouldBeValid: true,
		},
		{
			description: "universal CSS selector",
			// The universal selector. Probably won't be used, but we should
			// make sure it's valid so we capture the full selector spec
			// https://developer.mozilla.org/en-US/docs/Web/CSS/Universal_selectors
			value:         "*",
			shouldBeValid: true,
		},
		{
			description:   "tilde",
			value:         "div#mySpecialDiv.coolClass ~ span.anotherClass",
			shouldBeValid: true,
		},
		{
			description:   "missing argument to not()",
			value:         "div:not()",
			shouldBeValid: false,
		},
		{
			// This will still be treated as a single CSS selector, rather
			// than a group.
			description:   "alternating selector",
			value:         "div,span",
			shouldBeValid: true,
		},
		{
			// Can't be blank
			description:   "empty selector",
			value:         "",
			shouldBeValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			_, err := parseCSSSelector(tc.value)

			if v := err == nil; v != tc.shouldBeValid {
				t.Errorf("Unexpected error status for %v\nWanted: %v\nGot: %v\nError: %v", tc.value, tc.shouldBeValid, v, err)
			}
		})
	}

}

func TestCheckAndSetDefaults(t *testing.T) {
	cases := []struct {
		description        string
		input              Config
		expectErrSubstring string
	}{
		{
			description: "straightforward valid case",
			input: Config{
				Name:            "site-38911",
				URL:             mustParseURL("http://127.0.0.1:38911"),
				LinkSelector:    cascadia.MustCompile("a"),
				ItemSelector:    cascadia.MustCompile("ul li"),
				CaptionSelector: cascadia.MustCompile("p"),
			},
		},
		{
			description: "valid case with no link selector",
			input: Config{
				Name: "site-38911",
				URL:  mustParseURL("http://127.0.0.1:38911"),
			},
		},
		{
			description:        "item selector and caption selector but no link selector",
			expectErrSubstring: "link selector",
			input: Config{
				Name:            "site-38911",
				URL:             mustParseURL("http://127.0.0.1:38911"),
				ItemSelector:    cascadia.MustCompile("ul li"),
				CaptionSelector: cascadia.MustCompile("p"),
			},
		},
		{
			description:        "item selector and link selector but no caption selector",
			expectErrSubstring: "caption selector",
			input: Config{
				Name:         "site-38911",
				URL:          mustParseURL("http://127.0.0.1:38911"),
				ItemSelector: cascadia.MustCompile("ul li"),
				LinkSelector: cascadia.MustCompile("a"),
			},
		},
		{
			description: "no name",
			input: Config{
				URL:             mustParseURL("http://127.0.0.1:38911"),
				LinkSelector:    cascadia.MustCompile("a"),
				ItemSelector:    cascadia.MustCompile("ul li"),
				CaptionSelector: cascadia.MustCompile("p"),
			},
			expectErrSubstring: "name",
		},
		{
			description:        "no item selector",
			expectErrSubstring: "item selector",
			input: Config{
				Name:            "site-38911",
				URL:             mustParseURL("http://127.0.0.1:38911"),
				LinkSelector:    cascadia.MustCompile("a"),
				CaptionSelector: cascadia.MustCompile("p"),
			},
		},
		{
			description:        "no caption selector",
			expectErrSubstring: "caption selector",
			input: Config{
				Name:         "site-38911",
				URL:          mustParseURL("http://127.0.0.1:38911"),
				LinkSelector: cascadia.MustCompile("a"),
				ItemSelector: cascadia.MustCompile("ul li"),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			_, err := c.input.CheckAndSetDefaults()
			if c.expectErrSubstring != "" && err == nil {
				t.Fatalf(
					"expected an error with substring %v but got nil",
					c.expectErrSubstring,
				)
			}
			if c.expectErrSubstring != "" &&
				!strings.Contains(err.Error(), c.expectErrSubstring) {
				t.Fatalf(
					"expected error with substring %v but got %v",
					c.expectErrSubstring,
					err,
				)
			}
			if c.expectErrSubstring == "" && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
		})
	}
}

func TestCheckAndSetDefaultsWithZeroShortElementFilter(t *testing.T) {
	c := Config{
		Name:               "site-38911",
		URL:                mustParseURL("http://127.0.0.1:38911"),
		LinkSelector:       cascadia.MustCompile("a"),
		ItemSelector:       cascadia.MustCompile("ul li"),
		CaptionSelector:    cascadia.MustCompile("p"),
		ShortElementFilter: 0,
	}

	c2, err := c.CheckAndSetDefaults()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c2.ShortElementFilter != c.ShortElementFilter {
		t.Fatalf(
			"expected short element filter of %v but got %v",
			c.ShortElementFilter,
			c2.ShortElementFilter,
		)
	}
}
