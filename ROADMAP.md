# ROADMAP

## Doing now

## For the MVP (in order)
1. Retool the interface for `(sc *SMTPClient) Send(body string) error` in the `email` package. `Send()` both prepares an email and sends it using `dialer.DialAndSend(m)`. We can't--and don't need to--test `DialAndSend()`. Is there a way to use an interface that can allow for a test implementation of `DialAndSend()`?

2. Write e2e tests while refining the interfaces between packages as well as the glue code in `main.go`. Ensure that interfaces between packages are as small as possible.

3. Add log-based observability

4. Get to full unit test coverage

5. Handle more complex client situations in `poller.Poll()`, such as retries and non-2xx responses. (Currently this has no defensive measures at all.)

6. Add a message to the newsletter email when a link source's structure has changed and causes the provided selectors to return zero results.

7. Research some common web scraping pitfalls and ensure that this application avoids them.

8. Find/remove any bottlenecks. Can do this by creating profiles while running the e2e tests with flags on the `go test` command.

9. Update the README to describe the architecture of the application (what each package does and how they work together).

10. Make sure there's an adequate `doc.go` for every package.

11. **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.
