package linksrc

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	css "github.com/andybalholm/cascadia"
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
		return errors.New("the config must name the link source")
	}
	if n == "" {
		return errors.New("the link source name can't be blank")
	}
	c.Name = n

	if _, ok := v["url"]; !ok {
		return errors.New("the link source must include a URL")
	}

	u, err := parseURL(v["url"])
	if err != nil {
		return fmt.Errorf("can't parse the link source URL: %v", err)
	}
	c.URL = u

	var mi uint
	if _, mok := v["maxItems"]; !mok {
		mi = 5 // Set a low default so we don't accidentally process a ton of links
	} else {
		mii, err := strconv.Atoi(v["maxItems"])

		if err != nil || mii < 0 {
			return fmt.Errorf("invalid maxItems: must be a positive integer")
		} else {
			mi = uint(mii)
		}

		c.MaxItems = mi
	}

	// Check for the presence of an itemSelector, captionSelector, and
	// linkSelector. If there's only a linkSelector, we enable link auto-
	// detection. Otherwise, we need all three fields.

	if _, ok := v["itemSelector"]; ok {
		is, err := parseCSSSelector(v["itemSelector"])
		if err == nil {
			c.ItemSelector = is
		}
	}

	if _, ok := v["captionSelector"]; ok {
		cs, err := parseCSSSelector(v["captionSelector"])
		if err == nil {
			c.CaptionSelector = cs
		}
	}

	if _, ok := v["linkSelector"]; ok {
		ls, err := parseCSSSelector(v["linkSelector"])
		if err == nil {
			c.LinkSelector = ls
		}
	}

	if c.LinkSelector == nil {
		return errors.New("you must provide a link selector")
	}

	if (c.ItemSelector == nil && c.CaptionSelector != nil) ||
		(c.ItemSelector != nil && c.CaptionSelector == nil) {
		return errors.New("if you provide an item selector, you must provide a caption selector and vice versa")
	}

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
