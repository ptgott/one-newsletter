# One Newsletter

## A bento box for your media diet
One Newsletter checks online newspapers, magazines, and blogs for updates and emails you a newsletter with the latest links. You can then save these to a read-it-later service like Pocket, send them to friends, or whatever else you do with links to content. Unlike traditional RSS readers, you can limit the number of links you receive for each publication so you don't get overwhelmed. And since One Newsletter scrapes the sites you want to check, you're not limited to sites with RSS feeds.

## How is it deployed?
One Newsletter is designed to run on low-cost VMs (e.g., the least expensive [Digital Ocean VM](https://www.digitalocean.com/pricing/#standard-droplets)). Outside of the VM, the only required infrastructure is:

- **Persistent block storage:** One Newsletter keeps track of links it has already collected by storing them on disk via BadgerDB. You need to provide the path to a storage device that One Newsletter can use for BadgerDB's data directory.

- **An SMTP relay server:** One Newsletter needs to connect to an SMTP server in order to send email. This can be a service like Mailgun or a local relay like Postfix if you're into that sort of thing.

## How to run it
Run the following command:

```
onenewsletter -config path/to/config.yaml
```

One Newsletter reads its configuration from the YAML file at the `-config` path. The file has the following structure.

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

# This section configures the way One Newsletter scrapes websites for links.
# You must provide the interval at which One Newsletter checks for udpates and
# sends the newsletter, using a format like 5000ms, 5s, 10m, or 24h. To help
# prevent abuse, the minimum polling interval is 5s.
polling:
    interval: 168h # every seven days

# The storage section tells OneNewsletter how to store information abouts links
# it has already collected. "storageDir" is a path to a directory in which 
# One Newsletter will store its data via BadgerDB. "keyTTL" indicates how long
# each link will be stored in the database before it is deleted.
storage:
    storageDir: ./tempTestDir3012705204
    keyTTL: "168h"

# This section tells One Newsletter how to scrape websites for links. One
# Newsletter tracks these as "link sources." The assumption is that each 
# link source includes a menu of links (e.g., a "Most Read" list), and 
# One Newsletter scrapes these menus for updates.
#
# You must include a name for each link source, and the URL of a web page
# that includes a menu of links. You must also include the CSS selector of each
# list item in the menu (itemSelector). Next, you must include two CSS selectors 
# that are _relative to_ the itemSelector: one for the text that will accompany
# each link in the newsletter (captionSelector), and one for the HTML hyperlink
# reference (linkSelector).
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

```

## Testing
One Newsletter uses a mix of unit tests and end-to-end tests. Any new logic should be tested with unit tests unless it's not possible to do so. End-to-end tests live in the **e2e** directory. These run One Newsletter as a child process as well as an end-to-end testing process that includes separate goroutines for local HTTP and SMTP servers.

You can run all tests for One Newsletter from the project root with the following command:

```
go test ./...
```