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

		_, err := parseURL(tc.value)

		if v := err == nil; v != tc.shouldBeValid {
			t.Errorf("Unexpected error status for %v\nWanted: %v\nGot: %v\nError: %v", tc.value, tc.shouldBeValid, v, err)
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
		_, err := parseCSSSelector(tc.value)

		if v := err == nil; v != tc.shouldBeValid {
			t.Errorf("Unexpected error status for %v\nWanted: %v\nGot: %v\nError: %v", tc.value, tc.shouldBeValid, v, err)
		}
	}

}
