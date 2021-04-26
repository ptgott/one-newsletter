# ROADMAP

## For the MVP (in order)

1. Add a makefile with “test-unit” and “test-e2e” targets

1. Add inline `<style>` tags to the email HTML to achieve styling.

1. Consider making `main.go` less complex.

1. Handle more complex client situations when polling e-publications, such as retries and non-2xx responses. (Currently this has no defensive measures at all--we just call `http.Client.Get` in `main.go`)

1. Add more documentation:

   - Update the README to describe the architecture of the application (what each package does and how they work together).
   - Make sure there's an adequate `doc.go` for every package.

1. Add a message to the newsletter email when a link source's structure has changed and causes the provided selectors to return zero results.

1. Research some common web scraping pitfalls and ensure that this application avoids them.

1. Ensure this application's approach to logging isn't resource intensive. Consider turning more "info" logs into "debug" logs and configuring log output via CLI option. Also note that during some e2e tests, the application emits a ton of logs (particularly the `storing a link item in the database` logs). Maybe aggregate the `storing a link item` logs?

1. Consider using the lowest common ancestor approach shown [here](https://www.benawad.com/scraping-recipe-websites) to find lists of links automatically without requiring the user to specify this via a CSS selector.

1. Consider vendoring dependencies. Also consider running MailHog as a library (and vendoring it) so we don't have a dependency that becomes unavailable.

1. **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.
