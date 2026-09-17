# http-server

`http-server` builds instrumented HTTP servers for HTTP/1.1 and HTTP/2 service
traffic. It owns HTTP configuration, TLS termination, route middleware, and
serving on a caller-supplied listener.

## Installation

```bash
go get github.com/pbrpc/http-server
```

## Configuration

`FromEnv` reads these environment variables:

| Variable                   | Default  | Meaning |
| -------------------------- | -------- | ------- |
| `HTTP_SERVER_ADDRESS`      | `:50051` | TCP address exposed on the returned server |
| `HTTP_SERVER_IDLE_TIMEOUT` | `5m`     | Maximum time an idle connection is kept open |
| `HTTP2_SEND_PING_TIMEOUT`  | `2m`     | Quiet time before sending an HTTP/2 ping |
| `HTTP2_PING_TIMEOUT`       | `20s`    | Wait for a ping response before closing |
| `TLS_CERT`                 |          | Listener certificate as PEM |
| `TLS_KEY`                  |          | Listener private key as PEM |
| `TLS_CLIENT_CA`            |          | Client CA as PEM; enables verified client certificates |

## Server

The application creates the listener from the configured address and supplies
it to `Serve`:

```go
server, err := httpserver.FromEnv()
if err != nil {
	return err
}

server.Mux.Handle("/healthz", healthHandler)

listener, err := net.Listen("tcp", server.Address)
if err != nil {
	return err
}

return server.Serve(listener)
```

Every matched route runs under OpenTelemetry HTTP instrumentation. Route
middleware receives the matched mux pattern, with the first middleware passed
to `FromEnv` running outermost:

```go
server, err := httpserver.FromEnv(accessLog, streamDeadline)
```

TLS material selects HTTP/1.1 and HTTP/2 over TLS. Cleartext servers accept
HTTP/1.1 and unencrypted HTTP/2.
