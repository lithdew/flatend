# e2e

Make a single bootstrap node providing a `bootstrap` service, and have several other nodes provide a `node` service
which queries the `bootstrap` service every second.

```
$ DEBUG=* node bootstrap.js
  flatend Public Key: 071ed6b6ece42b31a4e0eaf4efd0689ae953c4a89f8672634fdbbf276df3c18a +0ms
  flatend Listening for Flatend nodes on '0.0.0.0:9000'. +6ms
No nodes were able to process your request for service(s): [node]
No nodes were able to process your request for service(s): [node]
No nodes were able to process your request for service(s): [node]
  flatend '0.0.0.0:40309' has connected to you. Services: [node] +3s
GOT 774 byte(s) from service ["node"].
GOT 774 byte(s) from service ["node"].
GOT 774 byte(s) from service ["node"].
GOT 774 byte(s) from service ["node"].

$ DEBUG=* node node.js
  flatend Public Key: 1ee37308010ef5d41a107dc50984edf6df5cd497dcc7e826ef829babd2b61558 +0ms
  flatend Listening for Flatend nodes on '0.0.0.0:40309'. +7ms
  flatend You have connected to '0.0.0.0:9000'. Services: [bootstrap] +63ms
  flatend Discovered 0 peer(s). +3ms
GOT 778 byte(s) from service ["bootstrap"].
GOT 778 byte(s) from service ["bootstrap"].
GOT 778 byte(s) from service ["bootstrap"].
GOT 778 byte(s) from service ["bootstrap"].

$ DEBUG=* node node.js
  flatend Public Key: bd6fc11af1f5b7956b810c439d74d9ca82abbae9847380c6b3dc4223a172f230 +0ms
  flatend Listening for Flatend nodes on '0.0.0.0:46329'. +9ms
  flatend You have connected to '0.0.0.0:9000'. Services: [bootstrap] +64ms
  flatend You have connected to '0.0.0.0:40309'. Services: [node] +63ms
  flatend Discovered 1 peer(s). +3ms
GOT 778 byte(s) from service ["bootstrap"].
GOT 778 byte(s) from service ["bootstrap"].
GOT 778 byte(s) from service ["bootstrap"].
GOT 778 byte(s) from service ["bootstrap"].
```

```js
const { Node, generateSecretKey } = require("flatend");
const { Readable } = require("stream");
const fs = require("fs");

const main = async () => {
  const node = await Node.start({
    secretKey: generateSecretKey(),
    bindAddrs: [`:9000`],
    services: {
      bootstrap: (ctx) => fs.createReadStream("bootstrap.js").pipe(ctx),
    },
  });

  setInterval(async () => {
    try {
      const stream = await node.push(["node"], {}, Readable.from([]));

      for await (const data of stream.body) {
        console.log(`GOT ${data.byteLength} byte(s) from service ["node"].`);
      }
    } catch (err) {
      console.error(err.message);
    }
  }, 1000);
};

main().catch((err) => console.error(err));
```

```js
const { Node, generateSecretKey } = require("flatend");
const { Readable } = require("stream");
const fs = require("fs");

const main = async () => {
  const node = await Node.start({
    secretKey: generateSecretKey(),
    addrs: [`:9000`],
    services: {
      node: (ctx) => fs.createReadStream("node.js").pipe(ctx),
    },
  });

  setInterval(async () => {
    try {
      const stream = await node.push(["bootstrap"], {}, Readable.from([]));

      for await (const data of stream.body) {
        console.log(
          `GOT ${data.byteLength} byte(s) from service ["bootstrap"].`
        );
      }
    } catch (err) {
      console.error(err.message);
    }
  }, 1000);
};

main().catch((err) => console.error(err));
```
