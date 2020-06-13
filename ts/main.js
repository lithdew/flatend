import nacl from "tweetnacl";
import {Flatend} from "./flatend.js";
import {ID} from "./packet.js";
import ip from "ip";


async function main() {
    const keys = nacl.sign.keyPair();
    const id = new ID({publicKey: keys.publicKey, host: "127.0.0.1", port: 12000});

    const backend = new Flatend({id, keys});

    backend.register("get_todos", data => {
        data = JSON.parse(data);

        if (parseInt(data?.params?.id) !== 123)
            throw new Error(`ID must be 123. Got ${data?.params?.id}.`);

        return data;
    });

    backend.register("all_todos", () => "hello world!");

    await backend.start({port: 9000, host: "127.0.0.1"});


    console.log(`Successfully connected to ${ip.toString(backend.id.host, 12, 4)}:${backend.id.port}.`);
}

main().catch(err => console.error(err));
