# One Newsletter

## A bento box for your media diet

One Newsletter checks online newspapers, magazines, and blogs for updates and
emails you a newsletter with the latest links. You can then save these to a
read-it-later service like Pocket, send them to friends, or whatever else you do
with links to content. 

Unlike traditional RSS readers, you can limit the number of links you receive
for each publication so you don't get overwhelmed. And since One Newsletter
scrapes the sites you want to check, you're not limited to sites with RSS feeds.

## How is it deployed?

One Newsletter is designed to run on low-cost VMs (e.g., the least expensive
[Digital Ocean VM](https://www.digitalocean.com/pricing/#standard-droplets)).
Outside of the VM, the only required infrastructure is:

- **Persistent block storage:** One Newsletter keeps track of links it has
 already collected by storing them on disk via BadgerDB. You need to provide
 the path to a storage device that One Newsletter can use for BadgerDB's data
 directory.

- **An SMTP relay server:** One Newsletter needs to connect to an SMTP server in
 order to send email. This can be a service like Mailgun or a local relay like
 Postfix if you're into that sort of thing.

## How to run it

Run the following command:

```
onenewsletter -config path/to/config.yaml
```

### Link sources and link items

One Newsletter works by scraping **link sources**, web pages with lists of links
to other web pages. These lists of links are called **link items**, and each one
is assumed to have both a link URL and a caption that describes the URL.

### Configuration

One Newsletter reads its configuration from the YAML file at the `-config` path.
The file has the following structure.

`email` configures the SMTP relay. The relay must advertise STARTTLS and AUTH.
One Newsletter negotiates a TLS connection and uses your username and pasword to
log in. Mutual TLS is currently not supported.

```yaml
email:
  smtpServerAddress: smtp://0.0.0.0:123
  fromAddress: mynewsletter@example.com
  toAddress: recipient@example.com
  username: MyUser123
  password: 123456-A_BCDE
```

`scraping` configures the scraper.

The `interval` field configures the way One Newsletter scrapes websites for
links. You must provide the interval at which One Newsletter checks for updates
and sends the newsletter, using a [Go duration
string](https://pkg.go.dev/time#ParseDuration) like `5000ms`, `5s`, `10m`, or
`24h`. To help prevent abuse, the minimum polling interval is `5s`. (Nothing is
stopping you from compiling One Newsletter with a lower interval, but please
don't be a jerk.)

`storageDir` is a path to a directory where One Newsletter stores its state. One
Newsletter keeps track of URLs it has already included in the newsletter so you
don't get repeat content. It stores URLs from the last two polling intervals.

`linkExpiryDays` indicates how many days One Newsletter will store the URLs of
links it has collected in the database. When One Newsletter collects a link, it
checks the link against the database to determine whether to email it to you.

```yaml
scraping:
  interval: 168h # every seven days
  storageDir: ./tempTestDir3012705204
  linkExpiryDays: 100
```

The `link_sources` section tells One Newsletter how to scrape websites for
links. One Newsletter tracks these as **link sources**. Each link source
includes a menu of links (e.g., a "Most Read" list), and One Newsletter scrapes
these menus for updates by examining the structure of its **link items**, i.e.,
a link and its surrounding HTML.

You can instruct One Newsletter to scrape links at three levels of specificity,
depending on how much you can tolerate unexpected results and you want to dig
into a website's CSS:

|Level|What you provide|What One Newsletter does|
|---|---|---|
|Fully automatic|The URL of a page, which can be an HTML page or an RSS/Atom feed|Identifies groups of link items and extracts captions based on the structure of each link item.|
|Automatic caption detection|The URL of a page and the CSS selector of a link within a given item (e.g., `ul li a`)|Extracts captions based on the structure of the identified links.|
|Fully manual|The URL of a page, the CSS selector of a link item, the CSS selector of a link within the item, and the CSS selector of a caption within the item.|Locates link items, links, and captions based on the provided information.|

A minimal link source configuration looks like this:

```yaml
link_sources:
  - name: site-1
    # HTML site
    url: https://www.example.com
  - name: site-2
    # RSS feed
    url: https://www.example.com/feed
```

Or to extract captions automatically but manually configure a link selector:

```yaml
link_sources:
  - name: site-1
    url: https://www.example.com
    linkSelector: "div#links article a"
```

Here is an example of a fully manual configuration. Let's say you want to follow
a website containing a menu of links that has the structure:

```html
<ul>
  <li>
    <p>Here is a caption</p>
    <p>You can find more information<a href="/cool-story">here</a>.
  <li>
  <li>
    <p>Here is another caption</p>
    <p>You can find more information<a href="/cool-story2">here</a>.
  <li>
</ul>
```

You can add this configuration:

```yaml
link_sources:
  - name: site-1
    url: https://www.example.com
    itemSelector: "ul li"
    captionSelector: "p"
    linkSelector: "a"
```

You can fine-tune the way One Newsletter includes links in emails.

`maxItems` specifies the maximum number of link items to include in an email for
a link source. The default is 5. If this is 0, One Newsletter will disregard it.
If more link items are found, One Newsletter won't extract links from them.

`minElementWords` is the minimum number of words that must be in a block-level
HTML element before we can add it to a link item's caption. This filters out
things like bylines, tags, and other text that doesn't display well in a
caption. 

It's hard to predict the kind of text that a site will include within an
element, so we set a pretty good default (three words) and enable users to
configure this. Set it to a lower value if a link source tends to include a lot
of two-word titles, for example.

Here is an example of a link source configuration with these fields:

```yaml
link_sources:
  - name: site-1
    url: https://www.example.com
    maxItems: 3
    minElementWords: 5
```

### Optional flags

By default, One Newsletter will periodically scrape the websites of your choice,
check the results against past results, and send an email containing the new
links. You can alter this behavior with the following flags:

- `-oneoff`: Carry out a single scrape and send a single email. Since One
  Newsletter only saves the results of a scrape in order to carry out repeated
  checks, this flag also stops it from saving results to the database. Useful if
  you want to try out One Newsletter in a "live" environment without waiting.

- `-test`: Print an email's HTML to standard output rather than sending it.
  Exits after the first email. You can then redirect the HTML to a file of your
  choice or just read it from the terminal. Useful for testing your
  configuration. Does not require any database or SMTP server configuration.

- `-level`: The level of logs to show. Can be `error`, `info`, `debug`, or
  `warn`. `info` by default. If you are using the `-test` flag, logging is
  disabled unless you specify a level.

### How automatic link item detection works

Automatic link item detection works from the assumption that each link sits in a
chunk of automatically generated HTML, e.g., the result of server-side template
rendering or client-side JavaScript components. We expect HTML around each link
to have a similar structure. Once you identify a link element, we can identify
that structure and extract captions from each repeating element.

To identify the link item that surrounds each `a` element, One Newsletter
traverses each `a` element's parents in the HTML element tree. It recursively
considers each parent until it identifies an HTML node that is (a) repeating and
(b) not identifical to itself. Each of these nodes becomes the root node in a
tree that will eventually contain a link item's caption.

Next, One Newsletter searches each link item's child nodes for possible
captions. For each child node, it extracts all of the text nodes below that
child node, and keeps track of those child nodes' immediate parents.

From there, it adds text nodes to the caption based on each text node's parent.
If a text node's parent is a block-level node, like a paragraph or a `div`, the
assumption is that the text is self contained. If the text in a block-level
element doesn't end in punctuation, One Newsletter adds a period. 

For text within inline nodes like `span`s, we assume that the text in the node
belongs to a wider whole, and append it to any neighboring inline elements.

Only block-level elements with more words than the user-configured
`minElementWords` end up in a link item's caption. We truncate each caption at
20 words. 

## Testing

One Newsletter uses a mix of unit tests and end-to-end tests. Any new logic
should be tested with unit tests unless it's not possible to do so. End-to-end
tests live in the **e2e** directory. These run One Newsletter as a child process
as well as an end-to-end testing process that includes separate goroutines for
local HTTP and SMTP servers.

You can run all tests for One Newsletter from the project root with the
following command:

```
go test ./...
```
