const {Node} = require("flatend");

const main = async () => {
    const node = new Node();
    node.register('hello_world', ctx => ctx.send("Hello world!"));
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));