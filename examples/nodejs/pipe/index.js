const {Node} = require("flatend");

const main = async () => {
    const node = new Node();
    node.register('pipe', ctx => ctx.pipe(ctx));
    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));