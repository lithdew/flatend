# count

Count from zero.

```
$ flatend
2020/06/17 02:12:53 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 02:12:53 Listening for HTTP requests on '[::]:3000'.
2020/06/17 02:12:59 ??? has connected to you. Services: [count]

$ node index.js 
Successfully dialed 127.0.0.1:9000. Services: []

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
const {Node} = require("flatend");

let counter = 0

const count = ctx => ctx.send(`${counter++}`);

const main = async () => {
    const node = new Node();
    node.register('count', count);
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));
```