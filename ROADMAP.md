# ROADMAP

## Required for deployment
- Benchmark memory/CPU usage while polling websites and sending an email. Make sure this is small enough to accommodate 1GB of RAM and 1vCPU (and with other stuff running on the same box).

## Can be done after deployment
1. Come up with a release process (i.e., to make it easier for people to use this without building from source). Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.

1. Add a Makefile with “test-unit” and “test-e2e” targets

1. Add more documentation:

   - Update the README to describe the architecture of the application (what each package does and how they work together).
   - Make sure there's an adequate `doc.go` for every package.

1. Consider using the lowest common ancestor approach shown [here](https://www.benawad.com/scraping-recipe-websites) to find lists of links automatically without requiring the user to specify this via a CSS selector.

1. Consider vendoring dependencies. Also consider running MailHog as a library (and vendoring it) so we don't have a dependency that risks becoming unavailable.

1. Add a unit test for `(db *BadgerDB) Cleanup()` so we don't need to rely on the e2e test.
