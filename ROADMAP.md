# ROADMAP

## Doing now

## For the MVP

- Reduce the number of exported linksrc members. Keep the interface small!

- Prevent missing config fields within `linksrc.Validate` (currently the test suite fails)

- Fill in more of `doc.go` within `linksrc`

- Grab HTML from user-selected sites at scheduled intervals

- Generate HTML to send as an email

- Email lists of links to the user

- Write e2e tests (also include profiles, which you can create using flags on the `go test` command)

- Add log-based observability

- **Releasing:** Change the module name to `www.github.com/ptgott/divnews`, including in all imports. Currently it's set to `divnews`.

## Possible features for after the MVP

- An SMTP server that receives email newsletters and filters them before forwarding to the user.
