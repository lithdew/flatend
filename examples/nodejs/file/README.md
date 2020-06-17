# file

Here's something trippy: a service that responds with its own source code.

```
$ flatend
2020/06/17 02:36:30 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 02:36:30 Listening for HTTP requests on '[::]:3000'.
2020/06/17 02:36:41 <anon> has connected to you. Services: [file]

$ DEBUG=* node index.js
  flatend You are now connected to 127.0.0.1:9000. Services: [] +0ms
  flatend Discovered 0 peer(s). +10ms

$ http://localhost:3000/
const { Node } = require("flatend");
const fs = require("fs");

const main = async () => {
  await Node.start({
    addrs: ["127.0.0.1:9000"],
    services: {
      file: (ctx) => fs.createReadStream("index.js").pipe(ctx),
    },
  });
};

main().catch((err) => console.error(err));
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /"
service = "file"
```

```js
const { Node } = require("flatend");
const fs = require("fs");

const main = async () => {
  await Node.start({
    addrs: ["127.0.0.1:9000"],
    services: {
      file: (ctx) => fs.createReadStream("index.js").pipe(ctx),
    },
  });
};

main().catch((err) => console.error(err));
```
