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

You can configure One Newsletter to detect link items within a link source in
two ways:

- Automatically: You identify the CSS selector for the `a` elements that that
 contain the links you want in your newsletter, and One Newsletter will
 identify the best caption for each link based on its surrounding HTML.
- Manually: You identify the CSS selector for each link item. Within each link
 item, you also identify the CSS selector for the caption and `a` element.

### Configuration

One Newsletter reads its configuration from the YAML file at the `-config` path.
The file has the following structure.

```yaml
# Configuration for the SMTP relay. The relay must advertise STARTTLS and
# AUTH. One Newsletter negotiates a TLS connection and uses your username
# and pasword to log in. Mutual TLS is currently not supported.
email:
  smtpServerAddress: smtp://0.0.0.0:123
  fromAddress: mynewsletter@example.com
  toAddress: recipient@example.com
  username: MyUser123
  password: 123456-A_BCDE

scraping:
  # The polling interval section configures the way One Newsletter scrapes
  # websites for links.
  # You must provide the interval at which One Newsletter checks for udpates and
  # sends the newsletter, using a format like 5000ms, 5s, 10m, or 24h. To help
  # prevent abuse, the minimum polling interval is 5s.
  interval: 168h # every seven days

  # The storage section tells OneNewsletter how to store information abouts links
  # it has already collected. "storageDir" is a path to a directory in which
  storageDir: ./tempTestDir3012705204

# This section tells One Newsletter how to scrape websites for links. One
# Newsletter tracks these as "link sources." The assumption is that each
# link source includes a menu of links (e.g., a "Most Read" list), and
# One Newsletter scrapes these menus for updates.
#
# To enable automatic link item detection, you only need to supply the name,
# URL, and linkSelector for each link source.
#
# To enable manual link item detection, you need to supply the name, URL,
# itemSelector, captionSelector, and linkSelector for each link source.
#
# The following example tracks a site with a link menu that has the structure,
# <ul>
#   <li>
#     <p>Here is a caption</p>
#     <p>You can find more information<a href="/cool-story">here</a>.
#   <li>
#   <li>
#     <p>Here is another caption</p>
#     <p>You can find more information<a href="/cool-story2">here</a>.
#   <li>
# </ul>
link_sources:
  - name: site-38911
    url: https://www.example.com
    itemSelector: "ul li"
    captionSelector: "p"
    linkSelector: "a"
    # Maximum number of link items to include in an email for a publication.
    # The default is 5. If this is 0, One Newsletter will disregard it.
    # If more link items are found, One Newsletter won't extract links from
    # them.
    maxItems: 10
    # The minimum number of words that must be in a block-level HTML element
    # before we can add it to a link item's caption. This filters out things
    # like bylines, tags, and other text that doesn't display well in a caption. 
    #
    # It's hard to predict the kind of text that a site will include
    # within an element, so we set a pretty good default (three words) and
    # enable users to configure this. Set it to a lower value if a link source
    # tends to include a lot of two-word titles, for example.
    minElementWords: 5
```

### Optional flags

By default, One Newsletter will periodically scrape the websites of your choice,
check the results against past results, and send an email containing the new
links. You can alter this behavior with the following flags:

- `-oneoff`: Carry out a single scrape and send a single email. Since One
 Newsletter only saves the results of a scrape in order to carry out repeated
 checks, this flag also stops it from saving results to the database. Useful if
 you want to try out One Newsletter without waiting.

- `-noemail`: Print an email's HTML to standard output rather than sending it.
 You can then redirect the HTML to a file of your choice or just read it from
 the terminal. Useful if you would like to run this on your local machine.

You can use the `-oneoff` and `-noemail` flags together for a quick
configuration check. One Newsletter will print the results of a scrape to your
terminal without sending an email.

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

Next, One Newsletter conducts a recursive, depth-first search of each link
item's child nodes for possible captions. For each child node, it extracts all
of the text nodes below that child node, and keeps track of those child nodes'
immediate parents.

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
