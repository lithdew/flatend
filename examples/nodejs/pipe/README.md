# pipe

Whatever comes in must come out. Simple piping example: upload a file to POST /pipe.

```
$ flatend
2020/06/17 01:07:12 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 01:07:12 Listening for HTTP requests on '[::]:3000'.
2020/06/17 01:07:17 <anon> has connected to you. Services: [pipe]

$ DEBUG=* node index.js
  flatend You are now connected to 127.0.0.1:9000. Services: [] +0ms
  flatend Discovered 0 peer(s). +19ms

$ POST /pipe (1.6MiB GIF)
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "POST /pipe"
service = "pipe"
```

```js
const { Node } = require("flatend");

const main = async () => {
  await Node.start({
    addrs: ["127.0.0.1:9000"],
    services: {
      pipe: (ctx) => ctx.pipe(ctx),
    },
  });
};

main().catch((err) => console.error(err));
```
