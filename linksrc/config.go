package linksrc

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	css "github.com/andybalholm/cascadia"
)

const (
	// Set a low default so we don't accidentally process a ton of links
	defaultMaxItems = 5

	// By default, we won't display one-word block element text, which looks
	// unattractive in captions.
	defaultMinElementWords = 3
)

// Config stores options for the link source container.
//
// There is no support for grouped (i.e., comma-separated) selectors. This is
// because, while grouped selectors are useful for applying styles to
// generalized sets of elements, the HTML parser needs to locate elements
// individually.
type Config struct {
	// The name of the source, e.g., "New York Magazine"
	Name string
	// url of the site containing links
	URL url.URL
	// CSS selector for a link within a list of links.
	ItemSelector css.Selector
	// CSS selector for a caption within a link item.
	// Relative to ItemSelector
	CaptionSelector css.Selector
	// CSS selector for the actual link within a link item. Should be an
	// "a" element. Relative to ItemSelector.
	LinkSelector css.Selector
	// Maximum number of Items in a Set. If a scraper returns more than this
	// within a link site, Items will be chosen arbitrarily.
	MaxItems uint
	// The minimum number of words that a block-level HTML element must
	// contain for it to be included in a link item's caption. Used to
	// exclude short pieces of text like blog tags, bylines, or anything
	// else that can get in the way of a caption's substance.
	//
	// Must be greater than zero. The default is three.
	ShortElementFilter int
}

// CheckAndSetDefaults validates c and either returns a copy of c with default
// settings applied or returns an error due to an invalid configuration
func (c *Config) CheckAndSetDefaults() (Config, error) {
	nc := *c

	if c.URL.String() == "" {
		return Config{}, errors.New("the link source must include a URL")
	}

	if c.Name == "" {
		return Config{}, errors.New("the link source name can't be blank")
	}

	if c.MaxItems <= 0 {
		nc.MaxItems = defaultMaxItems
	}

	// Check for the presence of an itemSelector, captionSelector, and
	// linkSelector. If there's only a linkSelector, we enable caption auto-
	// detection. If there is no link selector, we auto-detect links.
	// Otherwise, we need all three fields.
	if c.LinkSelector == nil && (c.ItemSelector != nil || c.CaptionSelector != nil) {
		return Config{}, errors.New("to detect captions manually, you must provide a link selector, item selector, and caption selector")
	}

	if (c.ItemSelector == nil && c.CaptionSelector != nil) ||
		(c.ItemSelector != nil && c.CaptionSelector == nil) {
		return Config{}, errors.New("if you provide an item selector, you must provide a caption selector and vice versa")
	}

	return nc, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface. Validation is
// performed here.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	v := make(map[string]string)
	err := unmarshal(&v)

	if err != nil {
		return fmt.Errorf("can't parse the email config: %v", err)
	}

	n, ok := v["name"]
	if !ok {
		n = ""
	}
	c.Name = n

	if _, ok := v["url"]; !ok {
		v["url"] = ""
	}

	u, err := parseURL(v["url"])
	if err != nil {
		return fmt.Errorf("can't parse the link source URL: %v", err)
	}
	c.URL = u

	var mi uint
	if _, mok := v["maxItems"]; !mok {
		mi = 0
	} else {
		mii, err := strconv.Atoi(v["maxItems"])

		if err != nil || mii < 0 {
			return fmt.Errorf("invalid maxItems: must be a positive integer")
		} else {
			mi = uint(mii)
		}

	}
	c.MaxItems = mi

	if _, ok := v["itemSelector"]; ok {
		is, err := parseCSSSelector(v["itemSelector"])
		if err != nil {
			return fmt.Errorf("cannot parse itemSelector: %v", err)
		}
		if err == nil {
			c.ItemSelector = is
		}
	}

	if _, ok := v["captionSelector"]; ok {
		cs, err := parseCSSSelector(v["captionSelector"])
		if err != nil {
			return fmt.Errorf("cannot parse captionSelector: %v", err)
		}
		if err == nil {
			c.CaptionSelector = cs
		}
	}

	if _, ok := v["linkSelector"]; ok {
		ls, err := parseCSSSelector(v["linkSelector"])
		if err != nil {
			return fmt.Errorf("cannot parse linkSelector: %v", err)
		}
		if err == nil {
			c.LinkSelector = ls
		}
	}

	var mt int
	if _, eok := v["minElementWords"]; !eok {
		// We need to set this when unmarshaling YAML, since otherwise
		// downstream consumers won't know if a zero value is
		// intentional.
		mt = defaultMinElementWords
	} else {
		mt, err = strconv.Atoi(v["minElementWords"])

		if err != nil || mt < 0 {
			return fmt.Errorf("invalid minElementWords: must be a positive integer")
		}

	}
	c.ShortElementFilter = mt
	return nil

}

// parseURL parses a URL for the purpose of defining home pages for
// link containers. We leave it to the caller to handle the validation errors.
func parseURL(s string) (url.URL, error) {
	u, err := url.Parse(s)

	if err != nil {
		return url.URL{}, err
	}

	// A URL like "http://#" will pass url.Parse despite being invalid.
	// In this case, url.Parse returns a u.String() that ends in a colon,
	// and a u.Scheme that doesn't.
	if strings.Replace(u.String(), ":", "", 1) == u.Scheme {
		return url.URL{}, fmt.Errorf("The URL %v is just a scheme", u.String())
	}

	if u.Scheme == "" {
		return url.URL{}, fmt.Errorf("the URL %v should include a scheme", u.String())
	}

	return *u, nil
}

// parseCSSSelector CSS selector strings into a type that's
// useful for locating HTML elements
func parseCSSSelector(s string) (css.Selector, error) {
	// Allowing groups of selectors since it's reasonable that a user
	// would want to find links within multiple wrapper elements on
	// the same website.
	c, err := css.Compile(s)

	if err != nil {
		return nil, err
	}

	return c, nil
}
