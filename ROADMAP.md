# ROADMAP

## Doing now

## Do later

1. Add a “-test” flag, which instructs the application to read the user config and open a sample email body in a local browser (or terminal), skipping any database reads/writes. This way, a user can make sure their config is correct without having to wait for the application to generate a newsletter. We can also add another flag for validating config without scraping.

1. Some sites, e.g., thekitchn.com, detect automation tools via the `User-Agent` header and refuse to show content if this is the case. Note that it’s often easy to bypass this restriction by using a `User-Agent` header copied from a legit Chrome request. But is that okay/legal?

1. Reduce the number of logs created when writing entries to the database. Currently there are four logs for every new link item we save, plus another log for adding the link item to an email.

1. Account for the possibility that some sites are dynamic. Maybe use a headless browser for all requests, rather than Go's HTTP client?

1. Provide an opt out option for users, to prevent mis-sent email issues! This probably means exposing an HTTP API.

1. Consider providing an option to update user configs via HTTP endpoint.

1. Come up with a release process (i.e., to make it easier for people to use this without building from source). Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.

1. Add a Makefile with “test-unit” and “test-e2e” targets. Also measure unit test coverage.

1. Consider using the lowest common ancestor approach shown [here](https://www.benawad.com/scraping-recipe-websites) to find lists of links automatically without requiring the user to specify this via a CSS selector.

1. Consider vendoring dependencies.

1. Add a unit test for `(db *BadgerDB) Cleanup()` so we don't need to rely on the e2e test.
