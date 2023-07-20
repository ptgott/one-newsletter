# Design Doc 1: Web UI for Remote Configuration

## The problem

To configure link sources in One Newsletter, a user needs to modify the
configuration file that the One Newsletter process reads, then restart the
process. 

Depending on how One Newsletter is deployed, it can get tedious to modify the
configuration. It also makes it difficult to decouple configuration from the
deployment, since on deployment, One Newsletter needs to have a configuration
file available to it.

## Possible approaches

### HTTP API with no web UI

Send YAML to the API as the body of an HTTP POST request. The tricky thing here,
other than the cumbersome nature of sending YAML over HTTP, is authentication.
Static bearer tokens or passwords are a no-go. It could be possible to generate
short-lived bearer tokens, but this would require another form of authentication
to fetch the latest token.

There are ways to make an HTTP API more secure, e.g., by requiring  a separate
header with a time-based one-time passcode (TOTP). The more complexity we add,
though, the less viable an HTTP API becomes.

### Client CLI

A client CLI, similar to `kubectl`, would handle authentication and wrap API
calls. You could apply configurations with a `newslectl app;ly`, for example.

The advantage of this approach is that there's no need for a graphical frontend.
It can work with the local filesystem, meaning that you can commit newsletter
configurations to your version control system and apply a configuration file
using Terraform.

The client binary can be the same as the server binary, making it possible to
test new newsletter configurations your workstation, then commit the changes to
the server.

Another upside is that the client CLI can speak gRPC with the server, which
automates some annoying parts of dealing with complex typed payloads like a big
configuration file. Of course, if we want to enable users to unsubscribe from a
newsletter by clicking on a link, we'll need to implement an HTTP API anyway, so
the gRPC route would require us to maintain two APIs.

The downside of using a client CLI for API calls is that it won't be possible to
adjust the newsletter configuration on mobile.

### HTTP API with a web UI

In this case, a web application would handle authentication with the backend,
serializing YAML, and POSTing configuration changes.

The advantage is that you could easily make configuration changes on mobile.

And while an early version would probably start with pasting YAML into a text
field, later iterations could mask the YAML with user-friendly UI components.
The Web UI could also handle validation, and could test the configuration in a
way that more accurately represents the final newsletter.

The disadvantge is that the web UI isn't friendly to version control.

### Some combination of the above

If the client CLI exists to make it esaire to authenticate to an HTTP API and
serialize YAML, it would, of course, be possible to support bare HTTP calls, a
web UI, and a CLI client. 

### Approach to start with

We can start with the CLI client because it's simpler than a web UI and more
user friendly than bare HTTP calls.

## Architecture

### Authentication

Let's use SSO with Google for authentication since it's simple and more secure
than anything I'd come up with!

#### Keeping this a single-tenant app

Add a field to the config file: `allow_email_addresses` (or similar): a list of
email addresses to allow to authenticate via Google.

#### Setting up SSO with Google

https://developers.google.com/identity/gsi/web/guides/overview

- "Sign in with Google is based on OAuth 2.0. The permissions that users granted
  through Sign in with Google are the same as those that they grant for OAuth,
  and vice versa."

- Note that there are separate APIs for (a) authenticating to Google and (b)
  obtaining information from a user's Google account. "To enforce this
  separation, the authentication API can only return ID tokens which are used to
  sign in to your website, whereas the authorization API can only return code or
  access tokens which are used only for data access but not sign-in."

https://developers.google.com/identity/gsi/web/guides/offerings

- You can choose from a One Tap button, which supports automatic sign-in, and a
  Sign In with Google button, which does not. In both cases, you can/should use
  Google's code generator to produce JavaSCript that consumes the appropriate
  API while following Google's guidelines.

Here's a link to the code generator that produces HTML to embed into your
website: https://developers.google.com/identity/gsi/web/tools/configurator

To set up SSO with Google, visit the Google APIs page, then obtain a client ID
and list authorized redirect URLs
(https://developers.google.com/identity/gsi/web/guides/get-google-api-clientid).
Part of the process involves configuring the Google consent screen
(https://developers.google.com/identity/gsi/web/guides/get-google-api-clientid#configure_your_oauth_consent_screen),
where you provide a logo for the app (as well as other information) to display
on Google's site.

When a user signs in via Google, Google sends an HTTP POST request to your login
endpoint
(https://developers.google.com/identity/gsi/web/guides/verify-google-id-token).
- Google uses the double-submit cookie pattern to prevent CSRF. 
- Use Google's public keys to verify Google's signature, making sure that you're
  using the latest public keys (Google rotates these regularly).
- Google recommends using its API client library
  (https://developers.google.com/identity/gsi/web/guides/verify-google-id-token#using-a-google-api-client-library)
  to validate its OAuth tokens.

Note that Google also has a general-purpose OIDC library. Here's a breakdown of
how to execute the "server flow" to authenticate a user:
https://developers.google.com/identity/openid-connect/openid-connect#server-flow
- Note that there's also an "implicit flow" that takes place in the browser.
  This is a more complicated alternative to the server flow. In this case,
  Google recomends using a Google Identity Services client library (see above).
- It looks like there's a Go library for handling OAuth 2.0 communication with
  Google APIs here: https://pkg.go.dev/golang.org/x/oauth2/google (double-check
  that I can implement the server flow described above this way)
- Note that an identity token's payload contains the email address of the
  authenticated user, so it should be straightorward to check this against the
  only allowable user if I add this as a configuration field.
  (https://developers.google.com/identity/openid-connect/openid-connect#an-id-tokens-payload)

### Answering ACME challenges

Which ACME challenge should we use?

Per Let's Encrypt, TLS-ALPN-01 is "is best suited to authors of TLS-terminating
reverse proxies that want to perform host-based validation like HTTP-01, but
want to do it entirely at the TLS layer in order to separate concerns"
(https://letsencrypt.org/docs/challenge-types/#tls-alpn-01)

Let's use HTTP-01, the most common challenge type. There's a Go library that
implements HTTP-01 (and other challenges:
https://pkg.go.dev/golang.org/x/crypto/acme)

### Using a CLI framework?

Now that the One Newsletter binary will support commands to run on both the
server and the client, we should handle the extra complexity (e.g., subcommands)
with a CLI framework.

Let's go with `cobra` since it's one of the oldest ones and a classic,
well-tested Go library with a big community. Looking at both `cobra` and
`alecthomas/kong`, a CLI library with a recent following, there didn't seem to
be enough of a difference to choose `kong` over its more venerable counterpart.

### Applying configurations

#### On the client side

An `apply` command takes a mandatory argument whose value is the path to a
configuration file. The command does the following:

- Initiates the Google OAuth 2.0 authentication flow and retrieves a token.
- Reads the `link_sources` section of named config file.
- Serializes the configuration data and signs the payload with the token.
- Sends the request to the backend, `POST`ing to `/api/v1/config/`.

#### On the server side

Configuration ingestion will involve a single API path, `/api/v1/config`, which
only supports the `POST` method. It:

- Verifies the token and that the token signed the payload,
  authenticating/authorizing the user.
- Uses sqlite to load a config file from disk if it already exists.
- Replaces the old config with the new config, writing it to disk via sqlite.

This will involve:

- Using sqlite
- Adding a server-side configuration option for where to put the sqlite
  database. Currently the only configuration option for a storage file path is
  `scraping.storageDir`.

It's much simpler to override all configuration options than to support more
sophisticating configuration merge logic. We can start with this approach, then
see what problems arise.
