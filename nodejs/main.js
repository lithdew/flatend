import {Flatend} from "./flatend.js";
import ip from "ip";

const getTodos = data => {
    data = JSON.parse(data);

    const id = parseInt(data?.params?.id);
    if (id !== 123) {
        throw new Error(`ID must be 123. Got ${id} instead.`);
    }

    return data;
}

async function main() {
    const backend = new Flatend();

    backend.register("all_todos", () => "hello world!");
    backend.register("get_todos", getTodos);

    await backend.start({port: 9000, host: "127.0.0.1"});

    console.log(`Successfully connected to ${ip.toString(backend.id.host, 12, 4)}:${backend.id.port}.`);
}

// async function main() {
//     const keys = nacl.sign.keyPair();
//     const id = new ID({publicKey: keys.publicKey, host: "127.0.0.1", port: 12000});
//
//     const backend = new Flatend({id, keys});
//
//     backend.register("all_todos", () => "hello world!");
//     backend.register("get_todos", getTodos);
//
//     await backend.start({port: 9000, host: "127.0.0.1"});
//
//     console.log(`Successfully connected to ${ip.toString(backend.id.host, 12, 4)}:${backend.id.port}.`);
// }

main().catch(err => console.error(err));
