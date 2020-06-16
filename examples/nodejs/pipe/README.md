# pipe

Whatever comes in must come out. Simple piping example: upload a file to POST /pipe.

```
$ flatend
2020/06/17 01:07:12 Listening for microservices on '127.0.0.1:9000'.
2020/06/17 01:07:12 Listening for HTTP requests on '[::]:3000'.
2020/06/17 01:07:17 ??? has connected to you. Services: [pipe]

$ node index.js
Successfully dialed 127.0.0.1:9000. Services: []

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
const {Node} = require("flatend");

const main = async () => {
    const node = new Node();
    node.register('pipe', ctx => ctx.pipe(ctx));
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));
```