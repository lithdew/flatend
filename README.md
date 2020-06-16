<p align="center">
  <a href="/"><img 
    src="https://lh3.googleusercontent.com/pw/ACtC-3c6eZvrCLM-wV5UkBn8JZVBf-C-lAJ7XmCLgX5Gz4tCdbhCtREUw_o2bsYIbibU1fCk5A43h_9dBSV7y9hwtv9iIifKVk6QkGEGXYV1E1Kd0jyH62k8zZBsbbT3JSSfGRYW660frbzTO0wtTR4FQECl=s599-no" 
    width="200" border="0" alt="flatend"></a>
</p>

<div align="center">

<a href="LICENSE"><img src="https://img.shields.io/apm/l/atomic-design-ui.svg?" alt="MIT License"></a>
<a href="https://discord.gg/HZEbkeQ"><img src="https://img.shields.io/discord/697002823123992617" alt="Discord Chat"></a>
<a href="https://sourcegraph.com/github.com/lithdew/flatend?badge"><img src="https://sourcegraph.com/github.com/lithdew/flatend/-/badge.svg" alt="Sourcegraph"></a>

</div>

<div align="center">

<a href="https://pkg.go.dev/github.com/lithdew/flatend"><img src="https://img.shields.io/github/go-mod/go-version/lithdew/flatend" alt="Go version"></a>
<a href="https://pkg.go.dev/github.com/lithdew/flatend"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&amp;logoColor=white&amp;style=flat-square" alt="go.dev reference"></a>
<a href="http://godoc.org/github.com/lithdew/flatend"><img src="https://godoc.org/github.com/lithdew/flatend?status.svg" alt="GoDoc"></a>

</div>

<div align="center">

<a href="https://www.npmjs.com/package/flatend"><img src="https://img.shields.io/npm/v/flatend.svg?style=flat" alt="NPM version"></a>
<a href="https://www.npmjs.com/package/flatend"><img src="https://img.shields.io/npm/dm/flatend.svg?style=flat" alt="NPM downloads"></a>
<a href="https://github.com/nodejs/security-wg/blob/master/processes/responsible_disclosure_template.md"><img src="https://img.shields.io/badge/Security-Responsible%20Disclosure-yellow.svg" alt="Security Responsible
Disclosure"></a>

</div>

**flatend** provides the [scaffolding](https://github.com/lithdew/flatend) and [glue](https://github.com/lithdew/monte) for you to build high-performance, [end-to-end encrypted](DESIGN.md#security-as-a-service), [production-ready](DESIGN.md#production-ready-from-the-start) backends with [*zero* vendor lock-in](DESIGN.md#zero-vendor-lock-in-and-barriers).

## Requirements

Go v1.14 or later.

or

Node.js v14 stable (14.4.0) or later.

At the time being, **flatend** only supports NodeJS and Go. Support for Python and Deno is planned: [Join our Discord server](https://discord.gg/HZEbkeQ) to learn more.

## First: Choose Your HTTP Server

Flatend as a protocol is framework-agnostic.

We provide a prepackaged binary containing a fast, scalable, highly-configurable production-ready HTTP server written in Go with LetsEncrypt support.

For convenience, you may directly [download](https://github.com/lithdew/flatend/releases) the binary for quickstart.

Or, you could [build from source](#build-from-source).

## Then: Quickstart

[Download](https://github.com/lithdew/flatend/releases) the Flatend binary for your platform.

Create a new `config.toml`, and paste:

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /hello"
service = "hello_world"
```


HTTPS support is available via [LetsEncrypt](https://letsencrypt.org/). The requirements for enabling HTTPS are that you:

1. have a domain registered, and
2. have ports 80 and 443 open and available.

Should you meet those requirements, modify your `config.toml` `[[http]]` block like so:

```toml
[[http]]
https = true
domain = "lithdew.net"
```

Run the command

```shell
$ ./flatend -c config.toml
```

Depending on the language used, choose the corresponding SDK:

* [Go](#go-sdk)
* [NodeJS](#typescript-sdk)

### Go SDK

Add `flatend` as a Go module.

```shell
$ go get github.com/lithdew/flatend
```

Write a function that describes how to handle incoming requests for the `hello_world` service.

```go
package main

import "github.com/lithdew/flatend"

func helloWorld(ctx *flatend.Context) {
    ctx.WriteHeader("Content-Type", "text/plain; charset=utf-8")
    ctx.Write([]byte("Hello world!"))
}
```

Register `helloWorld` as a handler for the service `hello_world` and connect to the HTTP server listening for microservices at `127.0.0.1:9000`.

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

Run the HTTP Server

```
go run ./cmd/flatend
```

Run the Go program
```
go run app.go
```

Visit `localhost:9000/hello`

Restart either the Go program or the HTTP server.

Notice the Go program automatically reconnects to the HTTP server and vice versa.

For advanced `ctx` and `config.toml` configuration, please refer to [API Docs](API.md).

### Typescript SDK

For the following quickstart guide, we will be using TypeScript. Feel free to use pure JavaScript ES6.

Add `flatend` as a dependency to your project using npm/yarn.

```shell
$ npm install flatend
```

Write a function that describes how we want to handle incoming requests for the service `hello_world`.

```typescript
import {Context} from "flatend";

const helloWorld = (ctx: Context) => {
    ctx.header("Content-Type", "text/plain; charset=utf-8");
    ctx.send("Hello world!");
}
```

Register `helloWorld` as a handler for the service `hello_world` and connect to the HTTP server listening for microservices at `127.0.0.1:9000`.

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

Run the HTTP Server

```
go run ./cmd/flatend
```

Run the NodeJS program
```
npm start
```

Visit `localhost:9000/hello`

Restart either the NodeJS program or the HTTP server.

Notice the NodeJS program automatically reconnects to the HTTP server and vice versa.

For advanced `ctx` and `config.toml` configuration, please refer to [API Docs](API.md).

## API Documentation

Please refer to the [API Documentation](API.md).

## Build from Source

Clone this repository.

Run
`go build ./cmd/flatend`

Execute the binary
`sudo ./flatend` 

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

### HTTP Server

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

What is the design principle for **flatend**?

* The design principles are outlined in the [design documentation](DESIGN.md).

## License

**flatend**, and all of its source code is released under the [MIT License](https://github.com/lithdew/flatend/blob/master/LICENSE).