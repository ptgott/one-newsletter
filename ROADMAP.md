# ROADMAP

## Doing now

## For the MVP (in order)

1. Write e2e tests while refining the interfaces between packages as well as the glue code in `main.go`. Ensure that interfaces between packages are as small as possible.

1. Add log-based observability

1. Get to full unit test coverage

1. Handle more complex client situations in `poller.Poll()`, such as retries and non-2xx responses. (Currently this has no defensive measures at all.)

1. Add a message to the newsletter email when a link source's structure has changed and causes the provided selectors to return zero results.

1. Research some common web scraping pitfalls and ensure that this application avoids them.

1. Find/remove any bottlenecks. Can do this by creating profiles while running the e2e tests with flags on the `go test` command.

1. Update the README to describe the architecture of the application (what each package does and how they work together).

1. Make sure there's an adequate `doc.go` for every package.

1. **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.
