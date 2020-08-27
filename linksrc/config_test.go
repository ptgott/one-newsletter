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
		testCase{
			value:         "www.example.com",
			shouldBeValid: false, // Should include a scheme
		},
		testCase{
			value:         "http://www.example.com/path",
			shouldBeValid: true,
		},
		testCase{
			value:         "example.com",
			shouldBeValid: false,
		},
		testCase{
			value:         "localhost",
			shouldBeValid: false,
		},
		testCase{
			// For testing etc.
			value:         "localhost:3000",
			shouldBeValid: true,
		},
		testCase{
			value: "http://www.example",
			// Technically, TLDs can resolve to IP addresses, though this is super rare.
			// See:
			// https://serverfault.com/questions/90737/how-the-heck-is-http-to-a-valid-domain-name
			shouldBeValid: true,
		},
		testCase{
			// The origin will be our own webserver, so no relative URLs
			value:         "/path",
			shouldBeValid: false,
		},
		testCase{
			// Can't be blank
			value:         "",
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
		testCase{
			value:         "div ul li:nth-of-type(3)",
			shouldBeValid: true,
		},
		testCase{
			// Spelling errors but the same character classes as a valid case
			value:         "duv il la:uth-of-type(3)",
			shouldBeValid: false,
		},
		testCase{
			value:         "123",
			shouldBeValid: false,
		},
		testCase{
			// You can include HTMl elements with arbitrary tag names
			// and match them with a selector like this one.
			value:         "blah",
			shouldBeValid: true,
		},
		testCase{
			// The universal selector. Probably won't be used, but we should
			// make sure it's valid so we capture the full selector spec
			// https://developer.mozilla.org/en-US/docs/Web/CSS/Universal_selectors
			value:         "*",
			shouldBeValid: true,
		},
		testCase{
			value:         "div#mySpecialDiv.coolClass ~ span.anotherClass",
			shouldBeValid: true,
		},
		testCase{
			// Missing argument to not()
			value:         "div:not()",
			shouldBeValid: false,
		},
		testCase{
			// Comma-separated groups of CSS selectors must not be valid here,
			// since the parser needs to establish a hierarchy from individual list
			// items to their captions, links, etc.
			value:         "div,span",
			shouldBeValid: false,
		},
		testCase{
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
		value         RawConfig
		shouldBeValid bool
	}

	cases := []testCase{
		// Canonical/valid case
		testCase{
			value: RawConfig{
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: true,
		},
		// Invalid URL
		testCase{
			value: RawConfig{
				URL:             "example",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
		},
		// Missing fields
		testCase{
			value: RawConfig{
				URL:          "http://www.example.com/path",
				LinkSelector: "div.wrapper ul li a",
			},
			shouldBeValid: false,
		},
		// Invalid WrapperSelector
		testCase{
			value: RawConfig{
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
		},
		// Invalid ItemSelector
		testCase{
			value: RawConfig{
				URL:             "http://www.example.com/path",
				ItemSelector:    "123",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
		},
		// Invalid CaptionSelector
		testCase{
			value: RawConfig{
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "456",
				LinkSelector:    "div.wrapper ul li a",
			},
			shouldBeValid: false,
		},
		// Invalid LinkSelector
		testCase{
			value: RawConfig{
				URL:             "http://www.example.com/path",
				ItemSelector:    "div.wrapper ul li",
				CaptionSelector: "div.wrapper ul li span",
				LinkSelector:    "1431",
			},
			shouldBeValid: false,
		},
	}

	for _, tc := range cases {
		_, err := Validate(tc.value)

		if v := err == nil; v != tc.shouldBeValid {
			t.Errorf("Unexpected validity status for %v\nWanted: %v\nGot: %v\nError: %v", tc.value, tc.shouldBeValid, v, err)
		}
	}
}
