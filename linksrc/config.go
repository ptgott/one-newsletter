package linksrc

import (
	"fmt"
	"net/url"
	"strings"

	css "github.com/andybalholm/cascadia"
)

// Config stores options for the link source container. It's designed for
// parsing JSON sent and received across API boundaries, and could include
// arbitrary user input!
type Config struct {
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

// config represents a validated configuration document fit for
// consumption elsewhere in the application. There is no support
// for grouped (i.e., comma-separated) selectors. This is because, while
// grouped selectors are useful for applying styles to generalized sets of
// elements, the HTML parser needs to locate elements individually.
// Since member types are specific to external packages used for
// implementation, we should keep this unexported.
type config struct {
	// url of the site containing links
	url url.URL
	// CSS selector for a link within a list of links.
	itemSelector css.Selector
	// CSS selector for a caption within a link item.
	// Relative to ItemSelector
	captionSelector css.Selector
	// CSS selector for the actual link within a link item. Should be an
	// "a" element. Relative to ItemSelector.
	linkSelector css.Selector
	// The original data used in creating the config
	original Config
}

// validate indicates whether a link source configuration is valid and
// returns an error otherwise. Since it just returns one error, there
// might be even more lurking unseen.
func validate(c Config) (config, error) {

	// To make it faster/easier to edit invalid config docs, we'll
	// try to return as many errors as we can in one go, rather than
	// force callers to play the call-and-response game.
	errs := []string{}

	// First, make sure all fields are accounted for. We'll use a map
	// so we don't need to reflect.
	fields := make(map[string]string)

	fields["URL"] = c.URL
	fields["ItemSelector"] = c.ItemSelector
	fields["CaptionSelector"] = c.CaptionSelector
	fields["LinkSelector"] = c.LinkSelector

	for k, v := range fields {
		if v == "" {
			errs = append(errs, fmt.Errorf(
				"the config does not provide a value for %v",
				k,
			).Error())
		}
	}

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
		return config{}, fmt.Errorf(
			"The configuration was invalid. %v",
			strings.Join(errs, ", "),
		)
	}

	return config{
		url:             u,
		itemSelector:    is,
		captionSelector: cs,
		linkSelector:    ls,
		original:        c,
	}, nil
}

// validateURL validates a URL for the purpose of defining home pages for
// link containers. We leave it to the caller to handle the validation errors.
func validateURL(s string) (url.URL, error) {
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
