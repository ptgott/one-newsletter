# ROADMAP

## Doing now

## For the MVP

- Maybe retool the interface for `(sc *SMTPClient) Send(body string) error` in the `email` package. `Send()` both prepares an email and sends it using `dialer.DialAndSend(m)`. We can't--and don't need to--test `DialAndSend()`. Is there a way to use an interface that can allow for a test implementation of `DialAndSend()`?

- Handle more complex client situations in `poller.Poll()`, such as retries and non-2xx responses. (Currently this has no defensive measures at all.)

- Add a message to the newsletter email when a link source's structure has changed and causes the provided selectors to return zero results.

- Ensure that interfaces between packages are as small as possible

- Research some common web scraping pitfalls and ensure that this application avoids them.

- Get to full unit test coverage

- Write e2e tests (also include profiles, which you can create using flags on the `go test` command). Do this while refining the interfaces between packages as well as the glue code in `main.go`.

- Find/remove any bottlenecks

- Add log-based observability

- Update the README to describe the architecture of the application (what each package does and how they work together).

- Make sure there's an adequate `doc.go` for every package.

- **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.
