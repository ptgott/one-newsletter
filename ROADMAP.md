# ROADMAP

## Doing now

## For the MVP (in order)

1. Get to full (or full-enough) unit test coverage

1. Consider using the lowest common ancestor approach shown [here](https://www.benawad.com/scraping-recipe-websites) to find lists of links automatically without requiring the user to specify this via a CSS selector.

1. Edit `poller.Client.Poll`:

   - This function reads one `io.Reader` into another, which seems pretty inefficient, since the result of `Poll` is read soon after by `NewSet` There shouldn't be a reason to create an intermediary buffer between fetching a response body `Reader` and parsing it for HTML. **Maybe merge `Client.Poll` and `linksrc.NewSet`**?
   - Also avoid `ReadFrom`, which is unbounded and doesn't watch the size of the response body. Use a `LimitReader` here.
   - Handle more complex client situations in `poller.Poll()`, such as retries and non-2xx responses. (Currently this has no defensive measures at all.)
   - We need to close the response body somehow

1. Consider using the lowest common ancestor approach shown [here](https://www.benawad.com/scraping-recipe-websites) to find lists of links automatically without requiring the user to specify this via a CSS selector.

1. Add inline `<style>` tags to the email HTML to achieve styling.

1. When closing a BadgerDB instance, the current approach doesn't return an error--making this easier to use with defer--but also panics instead of handling the error. Consider an alternative approach, or maybe retries.

1. Add a message to the newsletter email when a link source's structure has changed and causes the provided selectors to return zero results.

1. Research some common web scraping pitfalls and ensure that this application avoids them.

1. Find/remove any bottlenecks. Can do this by creating profiles while running the e2e tests with flags on the `go test` command.

1. Update the README to describe the architecture of the application (what each package does and how they work together).

1. Make sure there's an adequate `doc.go` for every package.

1. Consider vendoring dependencies.

1. **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.

1. (Possibly) allow for sending email to a remote SMTP server, which would require us to implement TLS within the SMTP client. Add an e2e test for this.
