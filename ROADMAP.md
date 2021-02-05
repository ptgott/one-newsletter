# ROADMAP

## Doing now

## For the MVP

- Maybe retool the interface for `(sc *SMTPClient) Send(body string) error` in the `email` package. `Send()` both prepares an email and sends it using `dialer.DialAndSend(m)`. We can't--and don't need to--test `DialAndSend()`. Is there a way to use an interface that can allow for a test implementation of `DialAndSend()`?

- Grab HTML from user-selected sites at scheduled intervals. Write a new package for this. It will probably involve taking a raw `Config` struct and validating it into an internal config object, similar to the way `linksrc` works but tailored to grabbing HTML. Note that if we go this route, we'll need to extract `Config` from `linksrc`. Maybe add `Config` validation in a way that doesn't transform the `Config` into a struct with parsing-specific members (i.e., from the `cascadia` library etc)?

- Ensure that interfaces between packages are as small as possible

- Get to full unit test coverage

- Write e2e tests (also include profiles, which you can create using flags on the `go test` command). Do this while refining the interfaces between packages as well as the glue code in `main.go`.

- Find/remove any bottlenecks

- Add log-based observability

- Update the README to describe the architecture of the application (what each package does and how they work together).

- Make sure there's an adequate `doc.go` for every package.

- **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.
