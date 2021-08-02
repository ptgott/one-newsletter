# ROADMAP

## Helping One Newsletter fetch links from all news sites

1. Extract link items where the outermost element in each item is the link itself (e.g., https://nymag.com/intelligencer/).

1. Parse online publications where links aren’t nested within link items, but where each element of a link item is positioned as the sibling of the other elements, without an outer wrapper (see: https://aldaily.com/articles-of-note/).

1. If the HTML element that contains link items also includes other elements, One Newsletter only extracts link items up until it reaches one of the other elements. An example is with 3 Quarks Daily (https://3quarksdaily.com/), where One Newsletter only extracts link items until the first ad.

1. Some sites, e.g., thekitchn.com, detect automation tools via the `User-Agent` header and refuse to show content if the header has an unacceptable value. Note that it’s often easy to bypass this restriction by using a `User-Agent` header copied from (for example) a legitimate Chrome request. But is that okay/legal?

1. Account for the possibility that some sites are dynamic. Maybe use a headless browser for all requests, rather than Go's HTTP client?

## Making user operations easier

1. Reload or modify the One Newsletter config without exiting and restarting One Newsletter. Consider providing an option to update user configs via HTTP API.

1. Consider using the lowest common ancestor approach shown [here](https://www.benawad.com/scraping-recipe-websites) to find lists of links automatically without requiring the user to specify this via a CSS selector.

1. Generate configurations automatically via a browser extension or bookmarklet.

1. Come up with a release process (i.e., let people install this without building from source).

## Making development easier

1. Add a Makefile with “test-unit” and “test-e2e” targets. Also measure unit test coverage in a Make task.

1. Update the third-party licenses table using a CI job (probably using GitHub Actions) and [`go-licenses`](https://github.com/google/go-licenses).

## Making the newsletter more useful

1. Fetch the first sentence of each article that will be included in a newsletter and add that after the caption, giving users more of an idea of what to expect from each link.

## Performance

1. Reduce the number of logs One Newsletter outputs without sacrificing visibility.

## Making accidental emails less bothersome

1. Provide an opt-out option for users to prevent mis-sent email issues. This probably means exposing an HTTP endpoint to stop emails to a particular address, plus an "unsubscribe" link in the newsletter email.
