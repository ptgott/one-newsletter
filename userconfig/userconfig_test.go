package userconfig

import (
	"bytes"
	"reflect"
	"testing"
)

func TestGenerateUntrusted(t *testing.T) {
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
    relayAddress: 0.0.0.0:123
    keyPath: ./tempTestDir3012705204/testKey.pem
    certPath: ./tempTestDir3012705204/testCert.pem
    username: myuser123
    password: myuser123
    fromAddress: mynewsletter@example.com
    toAddress: recipient@example.com
link_sources:

    - name: site-38911
      url: http://127.0.0.1:38911
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"

    - name: site-42869
      url: http://127.0.0.1:42869
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"

    - name: site-39917
      url: http://127.0.0.1:39917
      itemSelector: "ul li"
      captionSelector: "p"
      linkSelector: "a"

polling:
    interval: 4s
storage:
    storageDir: ./tempTestDir3012705204
    keyTTL: "168h"
    cleanupInterval: "10m"`,
		},
	}

	for _, tc := range testCases {
		b := bytes.NewBuffer([]byte(tc.conf))
		m, err := generateUntrusted(b)

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

	}

}
