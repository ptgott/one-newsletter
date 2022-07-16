# ROADMAP

## Doing now

1. Run e2e tests in a separate goroutine rather than a child process. 

  Use a minimal main.go file and extract most of the `main` function to a package that we can import in tests.

### Within this: now

- Other  `exec.Command` calls to replace:

  - `TestOneOffFlagWithNoEmailFlag`

  In these tests, we should call `scrape.StartLoop` with a particular config. 
  We can then remove the first `exec.Command` call from `TestMain`. Also use the
  new `createUserConfig` function.

- Remove the `go build` call from `TestMain`.

### Within this: next

1. Use a mock clock in e2e tests when calling `scrape.StartLoop` with a
   `time.Ticker` channel. This way, we can advance the clock manually to run
   tests, and don't need to wait as much. Changing our e2e tests to run
   in-process doesn't actually save much time, so we can make some real gains by
   using a mock clock.

2. The longest test by far is `TestDBCleanup`, which takes nearly a minute to
   run! After implementing the mock clock, let's see if there's another way to
   reduce the time it takes to run this.

## Helping One Newsletter fetch links from all news sites

1. Make it easier to fetch links from a news site with a varied layout, e.g.,
   one featured link, a couple of sub-featured links, and a list of other links.

1. Some sites, e.g., thekitchn.com, detect automation tools via the `User-Agent`
   header and refuse to show content if the header has an unacceptable value.
   Note that it’s often easy to bypass this restriction by using a `User-Agent`
   header copied from (for example) a legitimate Chrome request. But is that
   okay/legal?

1. Some sites use 301 redirects with cookies in order to conduct client
   classification (e.g.,
   https://www.imperva.com/blog/how-incapsula-client-classification-challenges-bots/).

One example is https://aldaily.com/articles-of-note, where the first request to
the page gets a 301 redirect containing a cookie and a `Location` header
pointing to the same path as before. The subsequent request uses the cookie.

Make the One Newsletter HTTP client more sophisticated so it passes client
classification tests (unless it genuinely shouldn't by some commonly accepted
standard). For example, we can set a `Jar` and `CheckRedirect` in the
`http.Client` (https://pkg.go.dev/net/http#Client).

1. Account for the possibility that some sites are dynamic. Maybe use a headless
   browser for all requests, rather than Go's HTTP client?

## Making user operations easier and safer

1. Make it easier to test new configurations.

- More helpful warnings about bad link selectors (e.g., not specific enough).
- Add verbose logs re: where in the automatic link parsing process One
- Newsletter failed to parse links. This would be useful for `oneoff`/`noemail`.
- Send the first email right away rather than after the scraping interval. This
  will make it easier to determine whether the app is running as expected.

1. Include help text when the CLI is run without arguments. Also add a `help` subcommand and flag.

1. Make it easier to test configurations locally.

   - Add a `-test` flag that combines `-oneoff` and `-noemail`.
   - For `-test`, don't include verbose logs.
   - For `-oneoff` and `-noemail` operations, don't touch the data directory
   - For `-test`, only read the `link_sources` part of the config.

1. Don't include verbose logs for `oneoff` operations.

1. Deprecate manual newsletter configuration (i.e., where you need to specify
   the link container, caption selector, and link selector).

1. Find an alternative to static passwords stored in the config file.

1. Reload or modify the One Newsletter config without exiting and restarting One
   Newsletter. Consider providing an option to update user configs via HTTP API.

1. Generate configurations automatically via a browser extension or bookmarklet.

1. Come up with a release process (i.e., let people install this without building from source).

## Making development easier

1. Add a Makefile with “test-unit” and “test-e2e” targets. Also measure unit test coverage in a make target.

1. Update the third-party licenses table using a CI job (probably using GitHub
   Actions) and [`go-licenses`](https://github.com/google/go-licenses). Or use a
   make target.


## Making the newsletter more useful

1. Fetch the first sentence of each article that will be included in a
   newsletter and add that after the caption, giving users more of an idea of
   what to expect from each link.

## Performance

1. Reduce the number of logs One Newsletter outputs without sacrificing visibility.

## Making accidental emails less bothersome

1. Provide an opt-out option for users to prevent mis-sent email issues. This
   probably means exposing an HTTP endpoint to stop emails to a particular
   address, plus an "unsubscribe" link in the newsletter email.
