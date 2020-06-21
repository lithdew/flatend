const { Node } = require("flatend");
const { Readable } = require("stream");

const main = async () => {
  const node = await Node.start({
    addrs: [`127.0.0.1:9000`],
    services: {
      hello_world: (ctx) => ctx.send("Hello world!"),
    },
  });

  setInterval(async () => {
    try {
      const stream = await node.push(
        ["hello_world"],
        {},
        Readable.from("hello world!")
      );

      for await (const data of stream) {
        console.log("GOT", data.toString("utf8"));
      }
    } catch (err) {
      console.error(err.message);
    }
  }, 1000);
};

main().catch((err) => console.error(err));
