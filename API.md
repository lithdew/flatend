## API Options

By default, Flatend comes prepackaged with a fast, scalable, highly-configurable production-ready HTTP server written in Go with [LetsEncrypt](https://letsencrypt.org/) support.

More prepackaged services are planned to be coming out soon from the Flatend team for creating production-ready WebSocket, TCP, and gRPC APIs out of Flatend services.

All of these prepackaged services are fully configurable with the list of configurable values specified below. Apart from configuring these services, services created using Flatend APIs may also be configured as shown below.

### Go

A Flatend node may be configured like so:

```go
import "github.com/lithdew/flatend"

node := &flatend.Node{
    // The public address to advertise to other nodes that may want to use this nodes' services.
    // Default is empty to indicate that the node does not advertise its services.
    PublicAddr: "...",

    // An Ed25519 secret key that uniquely identifies this node.
    SecretKey: kademlia.SecretKey{...},

    // Open this node up to other Flatend nodes on a specified TCP address.
    // Default is empty to indicate that this node only caters its services to other Flatend nodes
    // it manually connects to.
    BindAddrs: []flatend.BindFunc{
        flatend.BindAny(), // Randomly-selected open port.
        flatend.BindTCP(":3000"), // Specified IPv4/IPv6 address.
        flatend.BindTCPv4("127.0.0.1:3000"), // Specified IPv4 address.
        flatend.BindTCPv6("[::1]:3000"), // Specified IPv6 addresss
    }
}
```

### NodeJS

A Flatend node may be configured like so:

```typescript
import nacl from "tweetnacl";
import {Node, ID} from "flatend";

node := new Node({
    // Ed25519 public key, IPv4/IPv6 host, and unsigned 16-bit integer port.
    id: new ID(publicKey, host, port),
    
    // Ed25519 keypair that is unique to this node.
    keys: nacl.sign.keyPair(), 
})
```

## `ctx` Explained

### Go

Example
```go
import "github.com/lithdew/flatend"

func helloWorld(ctx *flatend.Context) {
    ctx.WriteHeader("Content-Type", "text/plain; charset=utf-8")
    ctx.Write([]byte("Hello world!"))
}
```

*Note*: Although `ctx *flatend.Context` implements `io.Writer`, and exposes the request body and headers as an `io.ReadCloser` and `map[string]string` respectively, do consider below:

- The headers associated to an incoming request may be accessed via `ctx.Headers`.
- The body of a request may be accessed via `ctx.Body`, which is an `io.ReadCloser`.
- It is advised to wrap the body with an `io.LimitedReader` as the length of the body of a request is unbounded.
- Upon the first call to `ctx.Write`, all response headers written via `ctx.WriteHeader` are dispatched to the requester. This implies that after the first write, no more headers may be written and dispatched to the requester.
- All data that is written is split and sent as encrypted chunks of 2048 bytes.
- The very moment the function returns, the response to a request is considered to be fully written.
- Any panics in a function are not caught.

### NodeJS

Example
```typescript
import {Context} from "flatend";

const helloWorld = (ctx: Context) => {
    ctx.header("Content-Type", "text/plain; charset=utf-8");
    ctx.send("Hello world!");
}
```

*Note*: `ctx: Context` is designed to be a NodeJS Duplex stream with a few extra properties and helper methods attached to it. A few rules to consider when writing functions in Flatend are:

- Upon the first write of response data towards request via a `Context`, all headers are dispatched to the requester. This implies that after the first write, no more headers may be set and dispatched to the requester.
- A handler must close a `Context`  to signal that a response has fully been written out by calling `ctx.end()`.
- All data that is written into `ctx` is split and sent as encrypted chunks of 2048 bytes.
- Streams, such as `fs.createFileStream(path: string)`, may be piped into a `ctx` as a response. 
- Any errors thrown in a handler are caught and sent as a JSON response to the requester.
- The headers associated to an incoming request may be accessed via `ctx.headers`.

The helper methods exposed in a `ctx: Context` are:

- `ctx.send(data: string | Uint8Array | Buffer)` writes `data` as a response, and closes `ctx`.
- `ctx.json(data: object)` encodes `data` into a JSON string, writes it as a response, and closes `ctx`.
- `await ctx.body({limit?: 65536})` reads the request body of `ctx`, with an optimal maximum size limit pre-configured to 65536 bytes. It throws an error if the size limit is exceeded.

## `config.toml` Explained

*Note*: As of v0.0.1, `config.toml` only supports `[[http]]` and address configuration.

```toml
[[http]]
https = true # Enable/disable HTTPS support. Default is false.

# Domain(s) for HTTPS support. Ignored if https = false.
domain = "lithdew.net"
domains = ["a.lithdew.net", "b.lithdew.net"]

# Addresses to serve HTTP requests on.
# Default is :80 if https = false, and :443 if https = true.

addr = ":3000"
addrs = [":3000", ":4000", "127.0.0.1:9000"]

# Remove trailing slashes in HTTP route path? Default is true.
redirect_trailing_slash = true

# Redirect to the exact configured HTTP route path? Default is true.
redirect_fixed_path = true

[http.timeout]
read = "10s" # HTTP request read timeout. Default is 10s.
read_header = "10s" # HTTP request header read timeout. Default is 10s.
idle = "10s" # Idle connection timeout. Default is 10s.
write = "10s" # HTTP response write timeout. Default is 10s.
shutdown = "10s" # Graceful shutdown timeout. Default is 10s.

[http.min]
body_size = 1048576 # Min HTTP request body size in bytes.

[http.max]
header_size = 1048576 # Max HTTP request header size in bytes.
body_size = 1048576 # Max HTTP request body size in bytes.

# The route below serves the contents of the file 'config.toml' upon
# recipient of a 'GET' request at path '/'. The contents of the file
# are instructed to not be cached to the requester.

# By default, caching for static files that are served is enabled.
# Instead of a file, a directory may be statically served as well.

[[http.routes]]
path = "GET /"
static = "config.toml"
nocache = true

# The route below takes an URL route parameter ':id', and includes it
# in a request sent to any Flatend node we know that advertises
# themselves of handling the service 'a', 'b', or 'c'. The HTTP
# request body, query parameters, and headers are additionally
# sent to the node.

[[http.routes]]
path = "POST /:id"
services = ["a", "b", "c"]
```