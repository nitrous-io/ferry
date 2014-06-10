# Ferry

A HTTP and WebSocket proxy for containers managed by a Freighter.

Name ideas:
* Ideas around the concept of boats piloting a ship into port: pilot, cutter,
  shore, navigator.
* Literal: proxyd.
* Others: mtgoxy, nigopro.

#### TODO

* HTTP Basic Auth (?).
* Map `host-name-6666.nitrousdomain.com` to port 6666 - get rid of forwarding
  port 80 to port 3000.
* Proxy TLS traffic so containers can use their own certs (only valid for custom
  domains).
* Proxy *any* traffic so that non-HTTP services (like Redis, databases) can be
  can be accessed.
* Should Ferry be Freighter-agnostic so that it works with any Docker daemon?
  Ferry -> an open source, generic reverse proxy to Docker containers. The
  Nitrous implementation would query Freighter to restrict to access to only
  Freighter-managed containers.

#### Building

1. Setup your [Golang build environment](http://golang.org/doc/code.html).
* Run `make install-deps`.
* Run `make`. This installs `ferry` into `$GOPATH/bin`.
