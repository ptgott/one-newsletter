package userconfig

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v2"
)

func TestParse(t *testing.T) {
	// Asserting deep equality between the expected and actual Meta would
	// be really convoluted and brittle, so we should make sure nothing
	// fails unexpectedly and test knottier marshaling/validation situations
	// elswhere.
	testCases := []struct {
		description   string
		conf          string
		shouldBeError bool
		shouldBeEmpty bool
	}{
		{
			description:   "valid case",
			shouldBeError: false,
			shouldBeEmpty: false,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"
scraping:
    interval: 5s
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "no email section",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf: `---
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"
scraping:
    interval: 5s
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "no link_sources section",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
scraping:
    interval: 5s
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "no scraping section",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"`,
		},
		{
			description:   "not yaml",
			shouldBeError: true,
			shouldBeEmpty: true,
			conf:          `this is not yaml`,
		},
		{
			description:   "no item selector or caption selector",
			shouldBeError: false,
			shouldBeEmpty: false,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
      linkSelector: "a"
scraping:
    interval: 5s
    storageDir: ./tempTestDir3012705204`,
		},
		{
			description:   "valid link source with no link selector",
			shouldBeError: false,
			shouldBeEmpty: false,
			conf: `---
email:
    smtpServerAddress: smtp://0.0.0.0:123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
    username: MyUser123
    password: 123456-A_BCDE
link_sources:
    - name: site-38911
      url: http://127.0.0.1:38911
scraping:
    interval: 5s
    storageDir: ./tempTestDir3012705204`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			b := bytes.NewBuffer([]byte(tc.conf))
			m, err := Parse(b)

			if (err != nil) != tc.shouldBeError {
				t.Errorf(
					"%v: unexpected error status: wanted %v but got %v with error %v",
					tc.description,
					tc.shouldBeError,
					err != nil,
					err,
				)
			}

			if reflect.DeepEqual(*m, Meta{}) != tc.shouldBeEmpty {
				l := map[bool]string{
					true:  "to be",
					false: "not to be",
				}
				t.Errorf(
					"%v: expected the Meta %v nil, but got the opposite",
					tc.description,
					l[tc.shouldBeEmpty],
				)
			}
		})

	}

}

func mustParseDuration(s string, t *testing.T) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestScrapingUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		description   string
		input         string
		shouldBeError bool
		expected      Scraping
	}{
		{
			description:   "valid case",
			shouldBeError: false,
			input: `storageDir: ./tempTestDir3012705204
interval: 5s
linkExpiryDays: 100`,
			expected: Scraping{
				Interval:       mustParseDuration("5s", t),
				StorageDirPath: "./tempTestDir3012705204",
				OneOff:         false,
				TestMode:       false,
				LinkExpiryDays: 100,
			},
		},
		{
			description:   "not an object",
			shouldBeError: true,
			input:         `[]`,
			expected:      Scraping{},
		},
		{
			description:   "unparseable duration",
			shouldBeError: true,
			input: `interval: 5y
storageDir: ./tempTestDir3012705204`,
			expected: Scraping{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var s Scraping
			empty := Scraping{}
			if err := yaml.NewDecoder(
				bytes.NewBuffer([]byte(tc.input)),
			).Decode(&s); (err != nil) != tc.shouldBeError {
				t.Errorf(
					"expected error status to be %v but got error %v",
					tc.shouldBeError,
					err,
				)
			}
			if tc.expected != empty {
				assert.Equal(t, tc.expected, s)
			}
		})
	}
}

func TestScrapingCheckAndSetDefaults(t *testing.T) {
	cases := []struct {
		description        string
		input              Scraping
		expected           Scraping
		expectErrSubstring string
	}{

		{
			description: "no storage path",
			input: Scraping{
				Interval: mustParseDuration("5s", t),
				OneOff:   false,
				TestMode: false,
			},
			expected:           Scraping{},
			expectErrSubstring: "path",
		},
		{
			description: "no interval",
			input: Scraping{
				OneOff:         false,
				TestMode:       false,
				StorageDirPath: "/storage",
			},
			expected:           Scraping{},
			expectErrSubstring: "interval",
		},
		{
			description: "interval less than 5s",
			input: Scraping{
				OneOff:         false,
				TestMode:       false,
				StorageDirPath: "/storage",
				Interval:       mustParseDuration("100ms", t),
			},
			expected:           Scraping{},
			expectErrSubstring: "5 seconds",
		},
		{
			description: "valid config with no link TTL",
			input: Scraping{
				OneOff:         false,
				TestMode:       false,
				StorageDirPath: "/storage",
				Interval:       mustParseDuration("10s", t),
			},
			expected: Scraping{
				Interval:       mustParseDuration("10s", t),
				StorageDirPath: "/storage",
				OneOff:         false,
				TestMode:       false,
				LinkExpiryDays: 180,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			actual, err := c.input.CheckAndSetDefaults()
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
			if !reflect.DeepEqual(actual, c.expected) {
				t.Fatalf("expected %+v but got %+v", c.expected, actual)
			}
		})
	}
}
