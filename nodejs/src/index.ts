import {Node, Context} from "./flatend";
import * as fs from "fs";

async function main() {
    const node = new Node();
    node.register("get_todos", (ctx: Context) => {
        ctx.write("hello world");
        ctx.end();
    });
    node.register("all_todos", (ctx: Context) => {
        const stream = fs.createReadStream("tsconfig.json");
        stream.pipe(ctx);
    });
    node.register('pipe', (ctx: Context) => ctx.pipe(ctx));
    await node.dial("0.0.0.0:9000");
}

main().catch(err => console.error(err));