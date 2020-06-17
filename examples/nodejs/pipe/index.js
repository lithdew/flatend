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
