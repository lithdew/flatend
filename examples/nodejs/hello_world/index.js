const {Node} = require("flatend");

const main = async () => {
    await Node.start({
        addrs: ["127.0.0.1:9000"],
        services: {
            "hello_world": ctx => ctx.send("Hello world!"),
        }
    });
}

main().catch(err => console.error(err));