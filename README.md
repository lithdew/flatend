# flatend

[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](LICENSE)
[![Discord Chat](https://img.shields.io/discord/697002823123992617)](https://discord.gg/HZEbkeQ)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lithdew/flatend)
[![npm version](https://img.shields.io/npm/v/flatend.svg?style=flat)](https://www.npmjs.com/package/flatend)
[![npm downloads](https://img.shields.io/npm/dm/flatend.svg?style=flat)](https://www.npmjs.com/package/flatend)
[![Security Responsible Disclosure](https://img.shields.io/badge/Security-Responsible%20Disclosure-yellow.svg)](https://github.com/nodejs/security-wg/blob/master/processes/responsible_disclosure_template.md)


<img align="right" width ="200" height="200" src="https://lh3.googleusercontent.com/pw/ACtC-3c6eZvrCLM-wV5UkBn8JZVBf-C-lAJ7XmCLgX5Gz4tCdbhCtREUw_o2bsYIbibU1fCk5A43h_9dBSV7y9hwtv9iIifKVk6QkGEGXYV1E1Kd0jyH62k8zZBsbbT3JSSfGRYW660frbzTO0wtTR4FQECl=s599-no">

**flatend** is an experimental framework and protocol to make microservices more modular, simpler, safer, cheaper, and faster to build using [p2p networking](https://github.com/lithdew/monte).

**flatend** aims to provide the benefits low-code tools try to bring to increase developer productivity, but with [zero vendor lock-in](https://news.ycombinator.com/item?id=20985429), [strong performance](https://projectricochet.com/blog/top-10-meteor-performance-problems), and [zero bias towards certain coding styles/patterns](https://news.ycombinator.com/item?id=12166666).

## Features

* Fully agnostic and compatible with any type of language, database, tool, library, or framework.
* P2P-based service discovery, load balancing, routing, and PKI via [Kademlia](https://en.wikipedia.org/wiki/Kademlia).
* Fully-encrypted, end-to-end, bidirectional streaming RPC via [Monte](https://github.com/lithdew/monte).
* Automatic reconnect/retry upon crashes or connection loss.
* Zero-hassle serverless: every function is a microservice.
* Stream multiple gigabytes of data across microservices.

## Gateways

**flatend** additionally comes with scalable, high-performance, production-ready, easily-deployable API gateways that are bundled into a [small, single executable binary](https://github.com/lithdew/flatend/releases) to help you quickly deploy your microservices.

* Written in [Go](https://golang.org/).
* HTTP/1.1, HTTP/2 support.
* Automatic HTTPS via [LetsEncrypt](https://letsencrypt.org/).
* Expose/load-balance across microservices.
* Serve static files and directories.
* REPL for real-time management (*coming soon!*).
* Prometheus metrics (*coming soon!*).
* WebSocket support (*coming soon!*).
* gRPC support (*coming soon!*).

All gateways have been extensively tested on [Rackspace](https://www.rackspace.com/), [Scaleway](https://www.scaleway.com/en/), [AWS](https://aws.amazon.com/), [Google Cloud](https://cloud.google.com/), and [DigitalOcean](https://www.digitalocean.com/).


## Requirements

Although **flatend** at its core is a protocol, and hence agnostic to whichever programming langauge you use, there are currently only two reference implementations in NodeJS and Go.

- NodeJS v12.18.1+ (Windows, Linux, Mac)
- Go v1.14.1 (Windows, Linux Mac)

The rationale for starting with NodeJS and Go is so that, for any new product/service, you may:

1. Quickly prototype and deploy in NodeJS with SQLite using a 2USD/month bare-metal server.
2. Once you start scaling up, split up your microservice and rewrite  the performance-critical parts in Go.
3. Run a red/blue deployment easily to gradually deploy your new microservices and experience zero downtime.

Support is planned for the following runtimes/languages:

1. [Zig v0.7+](https://ziglang.org/)
2. [Deno v1.0+](https://deno.land/)
3. [Python v3.8+](https://www.python.org/)

Have any questions? Come chat with us on [Discord](https://discord.gg/HZEbkeQ).

## Usage

To get started quickly, download the API gateway binary for your platform [here](https://github.com/lithdew/flatend/releases). Otherwise, build the binary from source by following the instructions [here](#build-from-source).

Create a new `config.toml`, and paste in:

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /hello"
service = "hello_world"
```

Run:

```shell
$ ./flatend
2020/06/18 04:07:07 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/18 04:07:07 Listening for HTTP requests on '[::]:3000'.
```

Now, let's build your first microservice in [Go](#go)/[NodeJS](#nodejs).

### Go

Add [`flatend`](https://pkg.go.dev/github.com/lithdew/flatend) to a new Go modules project.

```shell
$ go mod init github.com/lithdew/flatend-testbed
go: creating new go.mod: module github.com/lithdew/flatend-testbed

$ go get github.com/lithdew/flatend
go: downloading github.com/lithdew/flatend vX.X.X
go: github.com/lithdew/flatend upgrade => vX.X.X
```

Write a function that describes how to handle requests for the service `hello_world` in `main.go`.

```go
package main

import "github.com/lithdew/flatend"

func helloWorld(ctx *flatend.Context) {
    ctx.WriteHeader("Content-Type", "text/plain; charset=utf-8")
    ctx.Write([]byte("Hello world!"))
}
```

Register the function as a handler for the service `hello_world`.

```go
func main() {
    _ = &flatend.Node{
        Services: map[string]flatend.Handler{
            "hello_world": helloWorld,
        },
    }
}
```

Start the node and have it connect to Flatend's API gateway.

```go
func main() {
    node := &flatend.Node{
        Services: map[string]flatend.Handler{
            "hello_world": helloWorld,
        },
    }
    node.Start("127.0.0.1:9000")

    ch := make(chan os.Signal, 1)
    signal.Notify(ch, os.Interrupt)
    <-ch

    node.Shutdown()
}
```

Run it.

```shell
$ go run main.go
2020/06/18 04:09:25 Listening for Flatend nodes on '[::]:41581'.
2020/06/18 04:09:25 You are now connected to 127.0.0.1:9000. Services: []
2020/06/18 04:09:25 Re-probed 127.0.0.1:9000. Services: []
2020/06/18 04:09:25 Discovered 0 peer(s).
```

Visit [localhost:3000/hello](http://localhost:3000/hello).

```shell
$ curl http://localhost:3000/hello
Hello world!
```

Try restart your API gateway and watch your service re-discover it.

```shell
$ go run main.go 
2020/06/18 04:11:06 Listening for Flatend nodes on '[::]:39313'.
2020/06/18 04:11:06 You are now connected to 127.0.0.1:9000. Services: []
2020/06/18 04:11:06 Re-probed 127.0.0.1:9000. Services: []
2020/06/18 04:11:06 Discovered 0 peer(s).
2020/06/18 04:11:07 127.0.0.1:9000 has disconnected from you. Services: []
2020/06/18 04:11:07 Trying to reconnect to 127.0.0.1:9000. Sleeping for 500ms.
2020/06/18 04:11:08 Trying to reconnect to 127.0.0.1:9000. Sleeping for 617.563636ms.
2020/06/18 04:11:08 Trying to reconnect to 127.0.0.1:9000. Sleeping for 686.907514ms.
2020/06/18 04:11:09 You are now connected to 127.0.0.1:9000. Services: []
```

<p align="center">
    <img align="center" src="https://lh3.googleusercontent.com/pw/ACtC-3erxmTK_ro3rabuHCnaMGPgGDm0xx0GDhefsa30cR-dpSK-gSFCR4obN_PXiiXBjIodXHbVhMzuLZ9hR6ZJD7uq59sfflvDIJLdsk9lQjgAiMP1_6UhmCn9LJ6NmLf33Ts_zVfu7xeLs-2Pyk__jn4=w1000-h306-no" />
</p>

Check out more examples [here](https://github.com/lithdew/flatend/tree/master/examples/go). I recommend checking out the [Todo List](https://github.com/lithdew/flatend/tree/master/examples/go/todo) one which stores data in [SQLite](http://sqlite.org/).

### NodeJS

Add [`flatend`](https://www.npmjs.com/package/flatend) to a new npm/yarn project.

```shell
$ yarn init -y
yarn init vX.X.X
success Saved package.json

$ yarn add flatend
yarn add vX.X.X
info No lockfile found.
[1/4] Resolving packages...
[2/4] Fetching packages...
[3/4] Linking dependencies...
[4/4] Building fresh packages...

success Saved lockfile.
success Saved X new dependencies.
```

Write a function that describes how to handle requests for the service `hello_world` in `index.js`.

```js
const {Node, Context} = require("flatend");

const helloWorld = ctx => ctx.send("Hello world!");
```

Register the function as a handler for the service `hello_world`. Start the node and have it connect to Flatend's API gateway.

```js
const {Node, Context} = require("flatend");

const helloWorld = ctx => ctx.send("Hello world!");

async function main() {
    await Node.start({
        addrs: ["127.0.0.1:9000"],
        services: {
            'hello_world': helloWorld,
        },
    });
}

main().catch(err => console.error(err));
```

Run it.

```shell
$ DEBUG=* node index.js 
  flatend You are now connected to 127.0.0.1:9000. Services: [] +0ms
  flatend Discovered 0 peer(s). +19ms
```

Visit [localhost:3000/hello](http://localhost:3000/hello).

```shell
$ curl http://localhost:3000/hello
Hello world!
```

Try restart your API gateway and watch your service re-discover it.

```shell
$ DEBUG=* node index.js 
  flatend You are now connected to 127.0.0.1:9000. Services: [] +0ms
  flatend Discovered 0 peer(s). +19ms
  flatend Trying to reconnect to 127.0.0.1:9000. Sleeping for 500ms. +41s
  flatend Trying to reconnect to 127.0.0.1:9000. Sleeping for 500ms. +504ms
  flatend Trying to reconnect to 127.0.0.1:9000. Sleeping for 500ms. +503ms
  flatend Trying to reconnect to 127.0.0.1:9000. Sleeping for 500ms. +503ms
  flatend Trying to reconnect to 127.0.0.1:9000. Sleeping for 500ms. +503ms
  flatend You are now connected to 127.0.0.1:9000. Services: [] +21ms
```

<p align="center">
    <img src="https://lh3.googleusercontent.com/pw/ACtC-3eMjx0Nzshllz7xu2YIVUtFNwx4swwf8yZX56ojxXuGUv22z5c5a7NxoaRN4d2AUTdlQa7U3DEosJ_wdBBTSpT8ZSpuo3SbddZxOqO1B98ZsJ_B6rA3_H7bibw2KQKOkOtC64k3i_mJpWB3vfAvpeg=w800-h373-no" />
</p>

Check out more examples [here](https://github.com/lithdew/flatend/tree/master/examples/nodejs). I recommend checking out the [Todo List](https://github.com/lithdew/flatend/tree/master/examples/nodejs/todo) one which stores data in [SQLite](http://sqlite.org/).

## Options

### Go SDK

```go
package flatend

import "github.com/lithdew/kademlia"

type Node struct {
	// A reachable, public address which peers may reach you on.
	// The format of the address must be [host]:[port].
	PublicAddr string

	// A 32-byte Ed25519 private key. A secret key must be provided
	// to allow for peers to reach you. A secret key may be generated
	// by calling `flatend.GenerateSecretKey()`.
	SecretKey kademlia.PrivateKey

	// A list of IPv4/IPv6 addresses and ports assembled as [host]:[port] which
	// your Flatend node will listen for other nodes from.
	BindAddrs []string

	// A mapping of service names to their respective handlers.
	Services map[string]flatend.Handler

	// Total number of attempts to reconnect to a peer we reached that disconnected.
	// Default is 8 attempts, set to a negative integer to not attempt to reconnect at all.
	NumReconnectAttempts int

	// A factor proportionally representing how much larger each reconnection attempts
	// delay should increase by upon each attempt. Default is 1.25.
	ReconnectBackoffFactor float64

	// The minimum amount of time to wait before each reconnection attempt. Default is 500
	// milliseconds.
	ReconnectBackoffMinDuration time.Duration

	// The maximum amount of time to wait before each reconnection attempt. Default is 1
	// second.
	ReconnectBackoffMaxDuration time.Duration

    // ....
}

// Start takes in 'addrs', which is list of addresses to nodes to
// initially reach out for/bootstrap from first.
(*Node).Start(addrs string)

import "io"
import "io/ioutil"

func helloWorld(ctx *flatend.Context) {
    // The ID of the requester may be accessed via `ctx.ID`.
    _ = ctx.ID

    // All headers must be written before writing any response body data.
    
    // Headers are used to send small amounts of metadata to a requester.
    
    // For example, the HTTP API gateway directly sets headers provided
    // as a response as the headers of a HTTP response to a HTTP request
    // which has been transcribed to a Flatend service request that is
    // handled by some given node.

    ctx.WriteHeader("header key", "header val")

    // The first response body write call will send all set headers to the
    // requester. Any other headers set after the first call are ignored.
    ctx.Write([]byte("Hello world!"))


    // All request headers may be accessed via `ctx.Headers`. Headers
    // are represented as map[string]string.
    header, exists := ctx.Headers["params.id"]
    _, _ = header, exists

    // The body of a request may be accessed via `ctx.Body`. Request bodies
    // are unbounded in size, and represented as a `io.ReadCloser`.
    
    // It is advised to wrap the body under an `io.LimitReader` to limit
    // the size of the bodies of requests.

    buf, err := ioutil.ReadAll(io.LimitReader(ctx.Body, 65536))
    _, _ = buf, err

    // If no 'ctx.Write' calls are made by the end of the handler, an
    // empty response body is provided.
}
```

### NodeJS SDK

```js
const {Node} = require("flatend");

export interface NodeOptions {
  // A reachable, public address which peers may reach you on.
  // The format of the address must be [host]:[port].
  publicAddr?: string;

  // A list of [host]:[port] addresses which this node will bind a listener
  // against to accept new Flatend nodes.
  bindAddrs?: string[];

  // A list of addresses to nodes to initially reach out
  // for/bootstrap from first.
  addrs?: string[];

  // An Ed25519 secret key. A secret key must be provided to allow for
  // peers to reach you. A secret key may be generated by calling
  // 'flatend.generateSecretKey()'.
  secretKey?: Uint8Array;

  // A mapping of service names to their respective handlers.
  services?: { [key: string]: Handler };

  // Total number of attempts to reconnect to a peer we reached that disconnected.
  // Default is 8 attempts, set to 0 to not attempt to reconnect at all.
  numReconnectAttempts?: number;

  // The amount of time to wait before each reconnection attempt. Default is 500
  // milliseconds.
  reconnectBackoffDuration?: number;
}

await Node.start(opts: NodeOpts);

const {Context} = require("flatend");

// Handlers may optionally be declared as async, and may optionally
// return promises.

const helloWorld = async ctx => {
    // 'ctx' is a NodeJS Duplex stream. Writing to it writes a response
    // body, and reading from it reads a request body.

    _ = ctx.id; // The ID of the requester.

    ctx.pipe(ctx); // This would pipe all request data as response data.

    // Headers are used to send small amounts of metadata to a requester.
    
    // For example, the HTTP API gateway directly sets headers provided
    // as a response as the headers of a HTTP response to a HTTP request
    // which has been transcribed to a Flatend service request that is
    // handled by some given node.

    ctx.header("header key", "header val");

    // All request headers may be accessed via 'ctx.headers'. Headers
    // are represented as an object.

    // The line below closes the response with the body being a
    // JSON-encoded version of the request headers provided.

    ctx.json(ctx.headers);

    // Arbitrary streams may be piped into 'ctx', like the contents of
    // a file for example.

    const fs = require("fs");
    fs.createFileStream("index.js").pipe(ctx);

    // Any errors thrown in a handler are caught and sent as a JSON
    // response.

    throw new Error("This shouldn't happen!");

    // The 'ctx' stream must be closed, either manually via 'ctx.end()' or
    // via a function. Not closing 'ctx' will cause the handler to deadlock.

    // DO NOT DO THIS!
    // ctx.write("hello world!");

    // DO THIS!
    ctx.write("hello world!");
    ctx.end();

    // OR THIS!
    ctx.send("hello world!");

    // The line below reads the request body into a buffer up to 65536 bytes.
    // If the body exceeds 65536 bytes, an error will be thrown.

    const body = await ctx.read({limit: 65536});
    console.log("I got this message:", body.toString("utf8"));
};
```

### API Gateway

The configuration file for the API gateway is written in [TOML](https://github.com/toml-lang/toml).

```toml
# Address to listen for other Flatend nodes on.
addr = "127.0.0.1:9000"

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

## Build from source

```shell
$ git clone https://github.com/lithdew/flatend.git && cd flatend
Cloning into 'flatend'...
remote: Enumerating objects: 290, done.
remote: Counting objects: 100% (290/290), done.
remote: Compressing objects: 100% (186/186), done.
remote: Total 1063 (delta 144), reused 231 (delta 97), pack-reused 773
Receiving objects: 100% (1063/1063), 419.83 KiB | 796.00 KiB/s, done.
Resolving deltas: 100% (571/571), done.

$ go version
go version go1.14.4 linux/amd64

$ go build ./cmd/flatend
```

## Showcase

<a href="https://wars-mask.surge.sh/en" target="_blank"><img src="https://lh3.googleusercontent.com/pw/ACtC-3e7uWQGKd7_TV_cqSL89-CX15-fPr7bmfbSIbuteIlNgbqy2lLWr0ITNMmd-nVWin9tWwON-jpdjwWarFeAU3QyPDh4uKmcPBqa8PYoGPviIfGAS1XZ3bVvYnAoW7q3lYlEk9EQIvaLmoSMf4Z7_hTX=s465-no" width="80" height="auto"/></a>

[**Mask Demand Calculator**](https://wars-mask.surge.sh/en) - Helps you quickly calculate the amount of masks your household needs. Serving scraped RSS feeds with Flatend to more than 200K+ site visitors.

## Help

Got a question? Either:

1. Create an [issue](https://github.com/lithdew/flatend/issues/new).
2. Chat with us on [Discord](https://discord.gg/HZEbkeQ).

## FAQ

#### Is flatend production-ready? Who uses flatend today?

*flatend is still a heavy work-in-progress*. That being said, it is being field tested with a few enterprise projects related to energy and IoT right now.

Deployments of flatend have also been made with a few hundred thousand visitors.

#### Will I be able to run flatend myself?
   
It was built from the start to allow for self-hosting on the cloud, on bare-metal servers, in Docker containers, on Kubernetes, etc. The cloud is your limit (see the pun I did there?).

#### I'm worried about vendor lock-in - what happens if flatend goes out of business?

flatend's code is completely open in this single Github repository: there's no funny business going on here.

The mission of flatend is to eliminate vendor lock-in and be agnostic to any kinds of hosting environments starting from day one. Also to be somewhat of a breath of fresh air to the existing low-code tools out there.

#### How does flatend compare to `XXX`?

flatend gives me enough flexibility as a developer to use the tools and deployment patterns I want, gives me the scalability/performance I need, and at the same time lets me be very productive in building products/services quick.

flatend amalgamates a lot of what I sort of wish I had while building roughly tens of hackathon projects and startup projects.

For example, in many cases I just want to spend two bucks a month knowing that the things I build can easily handle a load of thousands of request per second.

Using the API gateways pre-provided with flatend, I can easily build a system that supports that and rapidly prototype its business logic in NodeJS.

#### Who owns the code that I write in flatend, and the data that I and my users save in flatend?

You own the data and the code. All the code is MIT licensed, and strongly compliant with GDPR/CCPA as well.

All communication across microservices are fully-encrypted end-to-end using AES-256 Galois Counter Mode (GCM). Encryption keys are ephemeral and established per-session, and are established using a X25519 Diffie-Hellman handshake followed by a single pass of BLAKE-2b 256-bit.

Y'know, basically just a hyper-specific standard configuration setting of the [Noise Protocol](http://www.noiseprotocol.org/).

#### I have a 3rd party/legacy system that I need to use with my backend. Can I still use flatend?

flatend from the start was made to be agnostic to whichever databases, programming languages, tools, or hosting environments you choose to put it through.

At the end of the day, flatend is just a protocol. That being said, to use flatend with your system would require writing a sort of shim or SDK for it.

Reach out to us on Discord, maybe the system you are looking to support may be an integration point well worth providing a reference implementation for.

## License

**flatend**, and all of its source code is released under the [MIT License](LICENSE).