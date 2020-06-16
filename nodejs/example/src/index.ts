import {Context, Node} from "flatend";
import * as fs from "fs";

async function main() {
    const node = new Node();

    node.register("get_todos", (ctx: Context) => ctx.json(ctx.headers));
    node.register("all_todos", (ctx: Context) => fs.createReadStream("./nodejs/package.json").pipe(ctx));

    node.register('pipe', async (ctx: Context) => {
        const body = await ctx.body({limit: 65536});
        ctx.json(JSON.parse(body.toString("utf8")));
    });

    await node.dial("0.0.0.0:9000");
}

main().catch(err => console.error(err));