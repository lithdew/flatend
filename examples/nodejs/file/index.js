const {Node} = require("flatend");
const fs = require("fs");

const main = async () => {
    const node = new Node();
    node.register('file', ctx => fs.createReadStream("index.js").pipe(ctx));
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));