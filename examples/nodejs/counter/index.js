const {Node} = require("flatend");

let counter = 0

const count = ctx => ctx.send(`${counter++}`);

const main = async () => {
    const node = new Node();
    node.register('count', count);
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));