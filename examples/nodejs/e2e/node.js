const { Node, generateSecretKey } = require("flatend");
const { Readable } = require("stream");
const fs = require("fs");

const main = async () => {
  const node = await Node.start({
    secretKey: generateSecretKey(),
    addrs: [`:9000`],
    services: {
      node: (ctx) => fs.createReadStream("node.js").pipe(ctx),
    },
  });

  setInterval(async () => {
    try {
      const stream = await node.push(["bootstrap"], {}, Readable.from([]));

      for await (const data of stream.body) {
        console.log(
          `GOT ${data.byteLength} byte(s) from service ["bootstrap"].`
        );
      }
    } catch (err) {
      console.error(err.message);
    }
  }, 1000);
};

main().catch((err) => console.error(err));
