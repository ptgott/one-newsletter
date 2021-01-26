package linksrc

import (
	"testing"
)

type testCase struct {
	value         string
	shouldBeValid bool
}

func TestValidateURL(t *testing.T) {

	cases := []testCase{
		{
			value:         "www.example.com",
			shouldBeValid: false, // Should include a scheme
		},
		{
			value:         "http://www.example.com/path",
			shouldBeValid: true,
		},
		{
			value:         "example.com",
			shouldBeValid: false,
		},
		{
			value:         "localhost",
			shouldBeValid: false,
		},
		{
			// For testing etc.
			value:         "localhost:3000",
			shouldBeValid: true,
		},
		{
			value: "http://www.example",
			// Technically, TLDs can resolve to IP addresses, though this is super rare.
			// See:
			// https://serverfault.com/questions/90737/how-the-heck-is-http-to-a-valid-domain-name
			shouldBeValid: true,
		},
		{
			// The origin will be our own webserver, so no relative URLs
			value:         "/path",
			shouldBeValid: false,
		},
		{
			// Can't be blank
			value:         "",
			shouldBeValid: false,
		},
		{
			// Just a hash with a scheme (definitely invalid but could
			// seem valid to logic that just checks for schemes)
			value:         "http://#",
			shouldBeValid: false,
		},
		{
			// Can't include a space
			value:         "http://www.example test.com",
			shouldBeValid: false,
		},
	}

	for _, tc := range cases {

		_, err := validateURL(tc.value)

		if v := err == nil; v != tc.shouldBeValid {
			t.Errorf("Unexpected validity status for %v\nWanted: %v\nGot: %v\nError: %v", tc.value, tc.shouldBeValid, v, err)
		}

	}
}

func TestValidateCSSSelector(t *testing.T) {

	cases := []testCase{
		{
			value:         "div ul li:nth-of-type(3)",
			shouldBeValid: true,
		},
		{
			// Spelling errors but the same character classes as a valid case
			value:         "duv il la:uth-of-type(3)",
			shouldBeValid: false,
		},
		{
			value:         "123",
			shouldBeValid: false,
		},
		{
			// You can include HTMl elements with arbitrary tag names
			// and match them with a selector like this one.
			value:         "blah",
			shouldBeValid: true,
		},
		{
			// The universal selector. Probably won't be used, but we should
			// make sure it's valid so we capture the full selector spec
			// https://developer.mozilla.org/en-US/docs/Web/CSS/Universal_selectors
			value:         "*",
			shouldBeValid: true,
		},
		{
			value:         "div#mySpecialDiv.coolClass ~ span.anotherClass",
			shouldBeValid: true,
		},
		{
			// Missing argument to not()
			value:         "div:not()",
			shouldBeValid: false,
		},
		{
			// This will still be treated as a single CSS selector, rather
			// than a group.
			value:         "div,span",
			shouldBeValid: true,
		},
		{
			// Can't be blank
			value:         "",
			shouldBeValid: false,
		},
	}

	for _, tc := range cases {
		_, err := validateCSSSelector(tc.value)

		if v := err == nil; v != tc.shouldBeValid {
			t.Errorf("Unexpected validity status for %v\nWanted: %v\nGot: %v\nError: %v", tc.value, tc.shouldBeValid, v, err)
		}
	}

}

func TestValidate(t *testing.T) {
	type testCase struct {
		value         Config
		shouldBeValid bool
		description   string // For logging
	}

	cases := []testCase{
		{
			value: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: true,
			description:   "canonical/valid case",
		},
		{
			value: Config{
				Name:            "Example Site",
				URL:             "example",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
			description:   "invalid URL",
		},
		{
			value: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com/path",
				ItemSelector:    "123",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
			description:   "invalid ItemSelector",
		},
		{
			value: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "456",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
			description:   "invalid CaptionSelector",
		},
		{
			value: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "1431",
			},
			shouldBeValid: false,
			description:   "invalid LinkSelector",
		},
		{
			value: Config{
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
			description:   "missing name",
		},
		{
			value: Config{
				Name:            "Example Site",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
			description:   "missing URL",
		},
		{
			value: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com/path",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
			description:   "missing item selector",
		},
		{
			value: Config{
				Name:         "Example Site",
				URL:          "http://www.example.com/path",
				ItemSelector: "div.wrapper ul li",
				LinkSelector: "div.wrapper ul li a",
			},
			shouldBeValid: false,
			description:   "missing caption selector",
		},
		{
			value: Config{
				Name:            "Example Site",
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
			},
			shouldBeValid: false,
			description:   "missing link selector",
		},
	}

	for _, tc := range cases {
		_, err := validate(tc.value)

		if v := err == nil; v != tc.shouldBeValid {
			t.Errorf("Unexpected validity status for %v\nWanted: %v\nGot: %v\nError: %v", tc.description, tc.shouldBeValid, v, err)
		}
	}
}
