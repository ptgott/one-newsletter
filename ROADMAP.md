# ROADMAP

## Doing now

## For the MVP

- Set up a Go module

- Grab HTML from user-selected sites at scheduled intervals

- Parse HTML into lists of links (Use this library? https://github.com/PuerkitoBio/goquery)

- Generate HTML to send as an email

- Email lists of links to the user

- Write e2e tests (also include profiles, which you can create using flags on the `go test` command)

## Possible features for after the MVP

- An SMTP server that receives email newsletters and filters them before forwarding to the user.
