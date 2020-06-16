<p align="center">
  <a href="/"><img 
    src="https://lh3.googleusercontent.com/pw/ACtC-3c6eZvrCLM-wV5UkBn8JZVBf-C-lAJ7XmCLgX5Gz4tCdbhCtREUw_o2bsYIbibU1fCk5A43h_9dBSV7y9hwtv9iIifKVk6QkGEGXYV1E1Kd0jyH62k8zZBsbbT3JSSfGRYW660frbzTO0wtTR4FQECl=s599-no" 
    width="200" border="0" alt="flatend"></a>
</p>

<div align="center">

[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](LICENSE)
[![Discord Chat](https://img.shields.io/discord/697002823123992617)](https://discord.gg/HZEbkeQ)
[![Sourcegraph](https://sourcegraph.com/github.com/lithdew/flatend/-/badge.svg)](https://sourcegraph.com/github.com/lithdew/flatend?badge)

</div>

<div align="center">

[![Go version](https://img.shields.io/github/go-mod/go-version/lithdew/flatend)](https://pkg.go.dev/github.com/lithdew/flatend)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lithdew/flatend)
[![GoDoc](https://godoc.org/github.com/lithdew/flatend?status.svg)](http://godoc.org/github.com/lithdew/flatend)

</div>

<div align="center">

[![NPM version](https://img.shields.io/npm/v/flatend.svg?style=flat)](https://www.npmjs.com/package/flatend)
[![NPM downloads](https://img.shields.io/npm/dm/flatend.svg?style=flat)](https://www.npmjs.com/package/flatend)
[![Security Responsible
Disclosure](https://img.shields.io/badge/Security-Responsible%20Disclosure-yellow.svg)](https://github.com/nodejs/security-wg/blob/master/processes/responsible_disclosure_template.md)

</div>

**flatend** is a tool and protocol for building high-performance, end-to-end encrypted, production-ready backends with zero vendor lock-in and hassle.

Write functions in your favorite language, using your favorite tools and platforms and libraries and databases.

**flatend** will turn them into production-ready microservices that are connected together and exposed as APIs using battle-tested p2p mesh networking, with just a few lines of code.

> "We should have some ways of connecting programs like garden hose--screw in
  another segment when it becomes necessary to massage data in
  another way. This is the way of IO also."
> Doug McIlroy. October 11, 1964

> "It's like low-code, but for developers without the vendor-lockin." Kenta Iwasaki. June 16, 2020

## Requirements
Go v1.14 or later.
Node.js v14 stable (14.4.0) or later.

At the time being, **flatend** only supports NodeJS and Go. Support for Python and Deno is planned: join our [Discord server](https://discord.gg/HZEbkeQ) to learn more.

## Table of Contents

- [Installation](#installation)
- [Quickstart](#quickstart)
    * [Go](#go)
    * [NodeJS](#nodejs)
- [Design](#design)
    * [Keep it simple, be flexible.](#keep-it-simple-be-flexible)
    * [Production-ready from the start.](#production-ready-from-the-start)
    * [Zero vendor lock-in and barriers.](#zero-vendor-lock-in-and-barriers)
    * [Security as a service.](#security-as-a-service)
- [Options](#options)
    * [Go](#go-1)
    * [NodeJS](#nodejs-1)
    * [HTTP Server](#http-server)
- [Showcase](#showcase)
- [Roadmap](#roadmap)
- [FAQ](#faq)
- [Help](#help)
- [License](#license)
## Installation

Head over to the [Releases](https://github.com/lithdew/flatend/releases) section and download the latest version of Flatend for your platform.

## Quickstart

Create a new `config.toml`, and paste:

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /hello"
service = "hello_world"
```

The configuration above will start a Flatend node that will advertise, service, and listen for other Flatend nodes at the address `127.0.0.1:9000`. The address must specify an explicit IP and port, as it will be used by other nodes that connect to your node as an integrity check.

The configuration above will also start up Flatend's pre-packaged production-ready HTTP server on port 3000, which will route HTTP requests to Flatend nodes that advertise themselves of being able to handle particular services by their name.

With the configuration above in this case, any GET requests to /hello will be forwarded to any other Flatend nodes that advertise themselves willing to handle the service `hello_world`.

HTTPS support is also available via [LetsEncrypt](https://letsencrypt.org/). The requirements for enabling HTTPS are that you:

1. have a domain registered, and
2. have ports 80 and 443 open and available.

Should you meet those requirements, modify your `config.toml` like so:

```toml
addr = "127.0.0.1:9000"

[[http]]
https = true
domain = "lithdew.net"

[[http.routes]]
path = "GET /hello"
service = "hello_world"
```

Afterwards, simply run the command below and watch your first Flatend node come to life:

```shell
$ ./flatend -c config.toml
```

Now, let's write our first Flatend microservice.

### Go

First, add `flatend` as a Go module into your project.

```shell
$ go get github.com/lithdew/flatend
```

Let's write a function that describes how we want to handle incoming requests for the service `hello_world`.

```go
package main

import "github.com/lithdew/flatend"

func helloWorld(ctx *flatend.Context) {
    ctx.WriteHeader("Content-Type", "text/plain; charset=utf-8")
    ctx.Write([]byte("Hello world!"))
}
```

In this case, we'll just reply to the request with "Hello world!".

Take note though that `ctx *flatend.Context` implements the `io.Writer`, and exposes the request body and headers as an `io.ReadCloser` and `map[string]string`  respectively. A few rules to consider when writing functions in Flatend are:

- The headers associated to an incoming request may be accessed via `ctx.Headers`.
- The body of a request may be accessed via `ctx.Body`, which is an `io.ReadCloser`.
- It is advised to wrap the body with an `io.LimitedReader` as the length of the body of a request is unbounded.
- Upon the first call to `ctx.Write`, all response headers written via `ctx.WriteHeader` are dispatched to the requester. This implies that after the first write, no more headers may be written and dispatched to the requester.
- All data that is written is split and sent as encrypted chunks of 2048 bytes.
- The very moment the function returns, the response to a request is considered to be fully written.
- Any panics in a function are not caught.

Now, we need just need to register `helloWorld` as a handler for the service `hello_world`, and hook it up to our HTTP server listening for microservices at `127.0.0.1:9000`.

```go
package main

import (
    "github.com/lithdew/flatend"
    "os"
    "os/signal"
)

func helloWorld(ctx *flatend.Context) {
    ctx.WriteHeader("Content-Type", "text/plain; charset=utf-8")
    ctx.Write([]byte("Hello world!"))
}

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

Run your Go program, visit `http://localhost:9000/hello` in your browser, and you should see "Hello world!". Tinker around restarting either the Go program or the HTTPS server, and notice the Go program automatically reconnecting to the HTTP server.

### NodeJS

For the following quickstart guide, we will be using TypeScript. However, any flavor of JavaScript will very much work here in principal, so feel free to use pure JavaScript ES6 as well for example.

First, add `flatend` as a dependency to your project using npm/yarn.

```shell
$ npm install flatend
```

Let's write a function that describes how we want to handle incoming requests for the service `hello_world`.

```typescript
import {Context} from "flatend";

const helloWorld = (ctx: Context) => {
    ctx.header("Content-Type", "text/plain; charset=utf-8");
    ctx.send("Hello world!");
}
```

In this case, we'll just reply to the request with "Hello world!".

Take note that `ctx: Context` is designed to be a NodeJS Duplex stream with a few extra properties and helper methods attached to it. A few rules to consider when writing functions in Flatend are:

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

Now, we need just need to register `helloWorld` as a handler for the service `hello_world`, and hook it up to our HTTP server listening for microservices at `127.0.0.1:9000`.

```typescript
import {Node, Context} from "flatend";

const helloWorld = (ctx: Context) => {
    ctx.header("Content-Type", "text/plain; charset=utf-8");
    ctx.send("Hello world!");
}

async function main() {
    const node = new Node();
    node.register("hello_world", helloWorld);
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));
```

Run your NodeJS program, visit `http://localhost:9000/hello` in your browser, and you should see "Hello world!". Tinker around restarting either the NodeJS program or the HTTPS server, and notice the NodeJS program automatically reconnecting to the HTTP server.

## Design

### Keep it simple, be flexible.

If you're like me, you dream of spending only 5$ a month to leave your product/service sustaining thousands of customers and requests per second. At this golden age of SaaS' and serverless and Docker containers and low-code tools and configuration hell, that dream is slowly starting to fade away.

Flatend uses the battle-tested p2p overlay networking protocol Kademlia to give you the building blocks necessary to achieve this dream with as much flexibility and as little code as possible. With Kademlia, microservices built with Flatend come with service discovery, load balancing, routing, and full industrial-grade end-to-end encryption built in.

All of that without the rigor of configuring Kubernetes/Istio/Consul over and over again. Of course though, if you love Kubernetes/Istio/Consul, then use it with Flatend. Run your Flatend services in Docker containers, on Kubernetes, or on AWS hassle-free.

Either way, Flatend is as minimal as it gets for you to be able to make the most out of your single 2$/month bare-metal server, or fleet of 10,000$/month cloud instances.

### Production-ready from the start.

At the end of the day, your microservices are just functions in Flatend.

Is your single behemoth microservice eating up your resources? Split it up into two functions and run one of the functions on another server without any downtime or networking hassle.

Running multiple projects and want to reuse your code? Package it up as yet another microservice with Flatend, and have it seamlessly interact with all of your projects.

As a matter of fact, one thing overly common is rewriting HTTP and WebSocket server/routing/middleware code over and over again. Flatend comes prepackaged with a fast, scalable, highly-configurable production-ready HTTP server written in Go with LetsEncrypt support.

### Zero vendor lock-in and barriers.

Flatend from the start was made to be _self-hosted_ and _open-source_. It was made to solve problems low-code tools are solving, without tying you to the bloodline of a newfound startup or company.

Flatend was also made to be completely agnostic to your devops, finances, frameworks, libraries, and preferences. Want to code with NodeJS? SQLite? kdb+? Go? QUIC? ORM X/Y/Z? You can use all of that with Flatend today.

It's agnostic and flexible to the point that you can serve a couple of your microservices on your laptop, and have your Flatend HTTP server on DigitalOcean route HTTP requests to them.

### Security as a service.

All communication across microservices written with Flatend is fully encrypted end-to-end using industrial-grade AES-256 Galois Counter Mode (GCM). The encryption key is established by a X25519 Diffie-Hellman handshake, whose results are passed through BLAKE-2b 256-bit.

Nonces used for encrypting/decrypting messages are unsigned big-endian 64-bit integers incremented per message. The only information exposed on each packet is a single prefixed 32-bit unsigned integer designating a packet's length. 

Microservices that are able to be discovered have public/private keys that are Ed25519. It is optional for microservices to identify themselves under a public key to provide services within a Flatend network.

The session handshake protocol is well-documented [here](https://github.com/lithdew/monte).

## Options

By default, Flatend comes prepackaged with a fast, scalable, highly-configurable production-ready HTTP server written in Go with LetsEncrypt support.

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

### HTTP Server

The pre-packaged HTTP server comes bundled with several configurable options, via `config.toml`:

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

## Showcase

<a href="https://wars-mask.surge.sh/en" target="_blank"><img src="https://lh3.googleusercontent.com/pw/ACtC-3e7uWQGKd7_TV_cqSL89-CX15-fPr7bmfbSIbuteIlNgbqy2lLWr0ITNMmd-nVWin9tWwON-jpdjwWarFeAU3QyPDh4uKmcPBqa8PYoGPviIfGAS1XZ3bVvYnAoW7q3lYlEk9EQIvaLmoSMf4Z7_hTX=s465-no" width="80" height="auto"/></a>

[**Mask Demand Calculator**](https://wars-mask.surge.sh/en) - An information site for calculating the masks your household needs. Serving RSS feeds with Flatend to more than 200K+ site visitors.

## Roadmap

### General

* Serve Flatend services as a gRPC API.
* Serve Flatend services as a WebSocket API.
* Experiment writing Flatend's pre-packaged servers in Ziglang.
* Multiplex HTTP/WebSocket/gRPC/Flatend services on a single port.
* Write Flatend SDK for:
  * Python
  * Deno
  * Ziglang
* Move bidirectional stream management to [lithdew/monte](https://github.com/lithdew/monte).

### HTTP

* Use [lithdew/boat](https://github.com/lithdew/boat) as a rule engine to conditionally allow/block request based on headers.
* Render Go/Markdown templates provided by a user.
* Support serving metrics and logs to a designated sink.
* Support proxying/redirecting to other routes/sites.
* Support alternative server packages like [valyala/fasthttp](https://github.com/valyala/fasthttp).
* Create a production-ready terminal REPL.
  * Serve terminal REPL as TTY/SSH/Web.
* Serve Prometheus metrics at an endpoint.
* Allow for re-ordering of middleware.
* Finer load balancing API.

### Go

- Finer load balancing API.
- Provide full support for Kademlia routing table.
- Reduce memory allocations in code hot-paths.

### NodeJS

- Finer load balancing API.
- Provide full support for Kademlia routing table.
- Submit requests to other Flatend services.
- Allow for graceful shutdown.
- Allow for custom loggers.

## Help

Got a question? Feed free to create an issue. Check out issues tagged with [question](https://github.com/lithdew/flatend/issues?q=is%3Aopen+is%3Aissue+label%3A%22question%22) first before creating a new issue.

## FAQ

Is **flatend** production-ready? Who should use **flatend** today?

* It is still a heavy work in progress, though **flatend** is currently being field-tested in a few enterprise projects for a startup and in a few other websites!

Will I be able to run **flatend** myself?

* Yes! **flatend** allows for self-hosting on the cloud, on bare metal servers, on Docker swarm; the sky is your limit.

I'm worried about vendor lock-in - what happens if **flatend** goes out of business?

* It doesn’t matter: **flatend**'s' source code has been public since day one and is agnostic to hosting environments.

Who owns the code that I write in **flatend**, and the data that I and my users save in **flatend**?

* You own the code you write in **flatend**, and you own your data. All transport is fully encrypted using industry-standard AES-256 GCM with a shared key derived from an X25519 Diffie-Hellman handshake.

Does **flatend** support testing?

* Yes!

I have a 3rd party and/or legacy systems that I need to use with my backend. Can I still use **flatend**?

* **flatend** is agnostic to databases, hosting solutions, and even programming languages, since at the end of the day, it is just a protocol. It definitely is compatible with whatever legacy or 3rd party systems you throw at it, provided you write the necessary support for it.

Are backends written in **flatend** compliant with GDPR/CCPA?

* Absolutely, yes. Just make sure the code you write with Flatend is compliant with GDPR/CCPA as well :).

## License

**flatend**, and all of its source code is released under the [MIT License](https://github.com/lithdew/flatend/blob/master/LICENSE).