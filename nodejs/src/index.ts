import {Node} from "./flatend";

async function main() {
    const node = new Node();
    node.register("get_todos", (data: any) => data);
    node.register("all_todos", () => "hello world");
    await node.dial("0.0.0.0:9000");
}

main().catch(err => console.error(err));