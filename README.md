# Ferry

A HTTP and WebSocket proxy for containers managed by a Freighter.

Name ideas:
* Ideas around the concept of boats piloting a ship into port: pilot, cutter,
  shore, navigator.
* Literal: proxyd.
* Others: mtgoxy, nigopro.

#### TODO

##### MVP

* Handle port 1024-9999.
* Forward port 80 to port 3000.
* Get host -> IP mapping from Freighter API.
* Stream Docker remote API's `GET /events` to receive updates of containers
  being stopped or started.
* Refresh host -> IP mapping from Freighter on Docker event.

##### Future

* HTTP Basic Auth (?).
* Map `host-name-6666.nitrousdomain.com` to port 6666 - get rid of forwarding
  port 80 to port 3000.
* Proxy TLS traffic so containers can use their own certs (only valid for custom
  domains).
* Proxy *any* traffic so that non-HTTP services (like Redis, databases) can be
  can be accessed.

#### Building

1. Setup your [Golang build environment](http://golang.org/doc/code.html).
*  Run `make`. This installs `ferry` into `$GOPATH/bin`.
