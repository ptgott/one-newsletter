package e2e

import "regexp"

// extractLinks takes a single email body and returns a slice of raw HTML link
// items. If an e2e test is failing and calls this function, make sure that the
// pattern it uses to match links is up to date.
func extractItems(body string) []string {
	if body == "" {
		return []string{}
	}
	linkPattern := regexp.MustCompile("<p>.*\\(<a href=\".*\">.*</a>\\)</p>")
	return linkPattern.FindAllString(body, -1)
}
