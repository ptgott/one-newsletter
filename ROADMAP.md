# ROADMAP

## Doing now

## For the MVP

- Maybe retool the interface for `(sc *SMTPClient) Send(body string) error` in the `email` package. `Send()` both prepares an email and sends it using `dialer.DialAndSend(m)`. We can't--and don't need to--test `DialAndSend()`. Is there a way to use an interface that can allow for a test implementation of `DialAndSend()`?

- Read email client config as well as link source config from user-provided data. (e.g., a JSON file)--find a good interface for this. Determine a secure way to get an SMTP server password from the user.

- Detect changes in link content against the previous scrape so we don't repeat content. We'll have access to a block storage volume to persist application state, though write an abstraction layer in case the method of storing data changes!

- Grab HTML from user-selected sites at scheduled intervals. Write a new package for this. It will probably involve taking a raw `Config` struct and validating it into an internal config object, similar to the way `linksrc` works but tailored to grabbing HTML. Note that if we go this route, we'll need to extract `Config` from `linksrc`. Maybe add `Config` validation in a way that doesn't transform the `Config` into a struct with parsing-specific members (i.e., from the `cascadia` library etc)?

- Write e2e tests (also include profiles, which you can create using flags on the `go test` command). Do this while refining the interfaces between packages as well as the glue code in `main.go`.

- Add log-based observability

- Make sure there's an adequate `doc.go` for every package.

- Get to full test coverage

- **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.
