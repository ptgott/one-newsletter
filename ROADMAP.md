# ROADMAP

## Doing now

## For the MVP

- Generate HTML to send as an email

- Grab HTML from user-selected sites at scheduled intervals. Write a new package for this. It will probably involve taking a raw `Config` struct and validating it into an internal config object, similar to the way `linksrc` works but tailored to grabbing HTML. Note that if we go this route, we'll need to extract `Config` from `linksrc`. Maybe add `Config` validation in a way that doesn't transform the `Config` into a struct with parsing-specific members (i.e., from the `cascadia` library etc)?

- Fill in more of `doc.go` within `linksrc`

- Email arbitrary HTML (passed from another package) to the user

- Write e2e tests (also include profiles, which you can create using flags on the `go test` command)

- Add log-based observability

- **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.

## Possible features for after the MVP

- An SMTP server that receives email newsletters and filters them before forwarding to the user.
