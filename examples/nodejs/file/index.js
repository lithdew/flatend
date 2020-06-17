const { Node } = require("flatend");
const fs = require("fs");

const main = async () => {
  await Node.start({
    addrs: ["127.0.0.1:9000"],
    services: {
      file: (ctx) => fs.createReadStream("index.js").pipe(ctx),
    },
  });
};

main().catch((err) => console.error(err));
