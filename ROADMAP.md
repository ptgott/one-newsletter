# ROADMAP

## Doing now

## For the MVP

- Set up a Go module

- Grab HTML from user-selected sites at scheduled intervals
  Add functions to create `WrapperMeta`s and and `ItemMeta`s from `Config`s (We should probably move these to another package, since the way we produce metadata from configs shouldn't be guaranteed between situations)

- Parse HTML into lists of links (Use this library? https://github.com/PuerkitoBio/goquery)

- Email lists of links to the user

- Write e2e tests (also include profiles, which you can create using flags on the `go test` command)
