# ROADMAP

## Doing now

## For the MVP (in order)

1. Add e2e tests related to data persistence:

   - Changing one link in an e-publication should result in an email with only that link
   - After the cleanup interval, the disk space used by the database should have reduced (otherwise, there may not be a good way to test `storage.Cleanup`)

1. Add a makefile with “test-unit” and “test-e2e” targets

1. Add inline `<style>` tags to the email HTML to achieve styling.

1. Handle more complex client situations when polling e-publications, such as retries and non-2xx responses. (Currently this has no defensive measures at all--we just call `http.Client.Get` in `main.go`)

1. Add more documentation:

   - Update the README to describe the architecture of the application (what each package does and how they work together).
   - Make sure there's an adequate `doc.go` for every package.

1. Add a message to the newsletter email when a link source's structure has changed and causes the provided selectors to return zero results.

1. Research some common web scraping pitfalls and ensure that this application avoids them.

1. Consider using the lowest common ancestor approach shown [here](https://www.benawad.com/scraping-recipe-websites) to find lists of links automatically without requiring the user to specify this via a CSS selector.

1. Consider vendoring dependencies.

1. **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.
