package linksrc

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	css "github.com/andybalholm/cascadia"
)

// RawConfig stores options for the link source container. It's designed for
// parsing JSON sent and received across API boundaries, and could include
// arbitrary user input!
type RawConfig struct {
	// URL of the site containing links
	URL string `json:"url"`
	// CSS selector for a link within a list of links
	ItemSelector string `json:"itemSelector"`
	// CSS selector for a caption, relative to ItemSelector
	CaptionSelector string `json:"captionSelector"`
	// CSS selector for the actual link within a link item. Should be an
	// "a" element. Relative to ItemSelector.
	LinkSelector string `json:"linkSelector"`
}

// Config represents a validated configuration document fit for
// consumption elsewhere in the application. There is no support
// for grouped (i.e., comma-separated) selectors. This is because, while
// grouped selectors are useful for applying styles to generalized sets of
// elements, the HTML parser needs to locate elements individually.
type Config struct {
	// URL of the site containing links
	URL url.URL
	// CSS selector for a link within a list of links.
	ItemSelector css.Selector
	// CSS selector for a caption within a link item.
	// Relative to ItemSelector
	CaptionSelector css.Selector
	// CSS selector for the actual link within a link item. Should be an
	// "a" element. Relative to ItemSelector.
	LinkSelector css.Selector
}

// Validate indicates whether a link source configuration is valid and
// returns an error otherwise. Since it just returns one error, there
// might be even more lurking unseen.
func Validate(c RawConfig) (Config, error) {

	errs := []string{}

	// TODO: Refactor all of the append statements into a function
	// so this is easier to read.

	u, err := validateURL(c.URL)

	if err != nil {
		errs = append(errs, err.Error())
	}

	is, err := validateCSSSelector(c.ItemSelector)

	if err != nil {
		errs = append(errs, err.Error())
	}
	cs, err := validateCSSSelector(c.CaptionSelector)

	if err != nil {
		errs = append(errs, err.Error())
	}

	ls, err := validateCSSSelector(c.LinkSelector)

	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf(
			"The configuration was invalid. %v",
			strings.Join(errs, ", "),
		)
	}

	return Config{
		URL:             u,
		ItemSelector:    is,
		CaptionSelector: cs,
		LinkSelector:    ls,
	}, nil
}

// validateURL validates a URL for the purpose of defining home pages for
// link containers. We leave it to the caller to handle the validation errors.
func validateURL(s string) (url.URL, error) {
	u, err := url.Parse(s)

	if err != nil {
		return url.URL{}, err
	}

	// It's not a host/port, so we're probably looking at domain name.
	if u.Scheme == "" {
		return url.URL{}, errors.New("the URL should include a scheme")
	}

	return *u, nil
}

// validateCSSSelector validates CSS selector strings
func validateCSSSelector(s string) (css.Selector, error) {
	// Allowing groups of selectors since it's reasonable that a user
	// would want to find links within multiple wrapper elements on
	// the same website.
	c, err := css.Compile(s)

	if err != nil {
		return nil, err
	}

	return c, nil
}
