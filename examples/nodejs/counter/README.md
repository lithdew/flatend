# count

Count from zero.

```
$ flatend
2020/06/17 02:12:53 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 02:12:53 Listening for HTTP requests on '[::]:3000'.
2020/06/17 02:12:59 <anon> has connected to you. Services: [count]

$ DEBUG=* node index.js
  flatend You are now connected to 127.0.0.1:9000. Services: [] +0ms
  flatend Discovered 0 peer(s). +14ms

$ http://localhost:3000
0

$ http://localhost:3000
1

$ http://localhost:3000
2
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /"
service = "count"
```

```js
const { Node } = require("flatend");

let counter = 0;

const count = (ctx) => ctx.send(`${counter++}`);

const main = async () => {
  await Node.start({
    addrs: ["127.0.0.1:9000"],
    services: { count: count },
  });
};

main().catch((err) => console.error(err));
```
