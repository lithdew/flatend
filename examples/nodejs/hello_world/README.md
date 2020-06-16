# hello_world

Create a service `hello_world` that replies with "Hello world!".

```
$ flatend
2020/06/17 00:44:34 Listening for microservices on '127.0.0.1:9000'.
2020/06/17 00:44:34 Listening for HTTP requests on '[::]:3000'.
2020/06/17 00:44:37 ??? has connected to you. Services: [hello_world]
2020/06/17 00:44:56 ??? has disconnected from you. Services: [hello_world]

$ http://localhost:3000/hello
no nodes were able to process your request for service(s): [hello_world]

$ node index.js
Successfully dialed 127.0.0.1:9000. Services: []

$ http://localhost:3000/hello
Hello world!
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /hello"
service = "hello_world"
```

```js
const {Node} = require("flatend");

const main = async () => {
    const node = new Node();
    node.register('hello_world', ctx => ctx.send("Hello world!"));
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));
```