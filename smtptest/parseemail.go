package smtptest

import "regexp"

// extractLinks takes a single email body and returns a slice of raw HTML link
// items. If an e2e test is failing and calls this function, make sure that the
// pattern it uses to match links is up to date.
func ExtractItems(body string) []string {
	if body == "" {
		return []string{}
	}
	linkPattern := regexp.MustCompile("<li>.*\\(<a href=\".*\">.*</a>\\)</li>")
	return linkPattern.FindAllString(body, -1)
}
