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
