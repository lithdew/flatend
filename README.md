# flatend

[![MIT License](https://img.shields.io/apm/l/atomic-design-ui.svg?)](LICENSE)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lithdew/flatend)
[![Discord Chat](https://img.shields.io/discord/697002823123992617)](https://discord.gg/HZEbkeQ)

Write functions in your favorite language, using your favorite tools and platforms and libraries and databases.

**flatend** will turn them into production-ready microservices that are connected together and exposed as APIs using battle-tested p2p mesh networking, with just a few lines of code.

**flatend** at the time being currently only supports NodeJS and Go. Support for Python and Deno is planned: join our Discord server to learn more.

> "We should have some ways of connecting programs like garden hose--screw in
  another segment when it becomes necessary to massage data in
  another way. This is the way of IO also."
> Doug McIlroy. October 11, 1964

> "It's like low-code, but for developers without the vendor-lockin." Kenta Iwasaki. June 16, 2020

## Design

### Keep it simple, be flexible.

If you're like me, you dream of spending only 5$ a month to leave your product/service sustaining thousands of customers and requests per second. At this golden age of SaaS' and serverless and Docker containers and low-code tools and configuration hell, that dream is slowly starting to fade away.

Flatend uses the battle-tested p2p overlay networking protocol Kademlia to give you the building blocks necessary to achieve this dream with as much flexibility and as little code as possible. With Kademlia, microservices built with Flatend come with service discovery, load balancing, routing, and fully-encrypted industrial-grade end-to-end encryption built in.

All of that without the rigor of configuring Kubernetes/Istio/Consul over and over again. Of course though, if you love Kubernetes/Istio/Consul, then use it with Flatend. Run your Flatend services in Docker containers, on Kubernetes, or on AWS hassle-free.

Either way, Flatend is as minimal as it gets for you to be able to make the most out of your single 2$/month bare-metal server, or fleet of 10,000$/month cloud instances.

### Production-ready from the start.

At the end of the day, your microservices are just functions in Flatend.

Is your single behemoth microservice eating up your resources? Split it up into two functions and run one of the functions on another server without any downtime or networking hassle.

Running multiple projects and want to reuse your code? Package it up as yet another microservice with Flatend, and have it seamlessly interact with all of your projects.

As a matter of fact, one thing I find overly common is rewriting HTTP and websocket server/routing/middleware code over and over again. Flatend comes pre-packaged with a fast, scalable, highly-configurable production-ready HTTP server written in Go with LetsEncrypt support.

### Zero vendor lock-in and barriers.

Flatend from the start was made to be _self-hosted_ and _open-source_. It was made to solve problems low-code tools are solving, without tying you to the bloodline of a newfound startup or company.

Flatend was also made to be completely agnostic to your devops, finances, frameworks, libraries, and preferences. Want to code with NodeJS? SQLite? kdb+? Go? QUIC? ORM X/Y/Z? You can use all of that with Flatend today.

It's agnostic to the point that you can serve your microservice on your laptop, and have your Flatend HTTP server on DigitalOcean route HTTP requests to your laptop.

### Security as a service.

All communication across microservices written with Flatend are fully-encrypted end-to-end using industrial-grade AES-256 Galois Counter Mode (GCM). The encryption key is established by a X25519 Diffie-Hellman handshake, whose results are passed through BLAKE-2b 256-bit.

Nonces used for encrypting/decrypting messages are unsigned big-endian 64-bit integers incremented per message. The only information exposed on each packet is a single prefixed 32-bit unsigned integer designating a packet's length. 

Microservices that are able to be discovered have public/private keys that are Ed25519. It is optional for microservices to identify themselves under a public key to provide services within a Flatend network.

The session handshake protocol is well-documented [here](https://github.com/lithdew/monte).

## Quickstart (NodeJS)

### Setup

Create a new `config.toml`, paste the following in, and run the command below.

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /hello"
service = "hello_world"
```

```shell
$ ./flatend -c config.toml
```

Based on the configuration above, Flatend will create a HTTP server listening on port 3000, serving a single route `GET /hello` which will route requests to the service named `hello_world`.

The configuration above will also have Flatend accept and transmit data to other Flatend microservices at 127.0.0.1:9000.

### Hello World

Now, let's write our first microservice.

For the following steps I will be using TypeScript, though use whatever flavor of JavaScript you prefer.

Let's write a function that describes how we want to handle incoming requests for the service `hello_world`.

```typescript
import {Context} from "flatend";

const helloWorld = (ctx: Context) => ctx.send("Hello world!");
```

In this case, we'll just reply to the request with "Hello world!". Take note that `ctx` in this case is a NodeJS Duplex stream which you may pipe data into and out of, with header data from our HTTP microservice accessible at `ctx.headers`.

Now, we need just need to register `helloWorld` as a handler for the service `hello_world`, and hook it up to our HTTP server listening for microservices at 127.0.0.1:9000.

```typescript
import {Node, Context} from "flatend";

const helloWorld = (ctx: Context) => ctx.send("Hello world!");

async function main() {
    const node = new Node();
    node.register("hello_world", helloWorld);
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));
``` 

Run your NodeJS program, visit `http://localhost:9000/hello` in your browser, and you should see "Hello world!".

There you have it; wasn't that easy :).