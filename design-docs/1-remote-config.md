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

<!--TODO: Handling HTTPS: how to do ACME stuff-->
<!--TODO: Handling authentication via the CLI app: how to get TOTP working-->
<!--TODO: Specific API paths to write-->
<!--TODO: How to handle consistency/versioning of the configuration. I.e.,
should there be a timestamp field/should we save a last known version UUID of
the configuration and compare that to the one of a recently applied config? Or
just accept whatever config gets applied?-->
<!--TODO: any other gotchas to look out for?-->
<!--TODO: use a CLI framework like Cobra (or others) now that the CLI is getting
more complex?-->
<!--TODO: What's the MVP?-->
