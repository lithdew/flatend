import nacl from "tweetnacl";
import {EventEmitter} from "events";
import MonteSocket from "./monte.js";
import {HandshakePacket, ID, RequestPacket, ResponsePacket} from "./packet.js";
import ip from "ip";

const OPCODE_HANDSHAKE = 0;
const OPCODE_REQUEST = 1;


class Flatend {
    /**
     *
     * @param {ID} [id]
     * @param {nacl.SignKeyPair} [keys]
     */
    constructor({id, keys}) {
        this.listeners = new EventEmitter();
        this.self = id;
        this.keys = keys;
    }

    /**
     *
     * @param {string} service
     * @param handler
     */
    register(service, handler) {
        if (this.listeners.listenerCount(service) > 0)
            throw new Error(`Service '${service}' is already registered.`)

        this.listeners.on(service, ({seq, data}) => {
            let res;
            try {
                res = handler(data);
            } catch (err) {
                res = {error: err.toString()}
            }

            if (!Buffer.isBuffer(res) && typeof res !== "string") {
                res = JSON.stringify(res);
            }

            this.conn.reply(seq, new ResponsePacket(Buffer.from(res)).encode());
        });
    }

    async start(opts) {
        this.conn = await MonteSocket.dial(opts);

        await this.handshake();

        this.conn.on('data', this._data.bind(this));
        this.conn.on('error', console.error);

        this.conn.on('close', () => {
            const reconnect = async () => {
                console.log(`Trying to reconnect to ${ip.toString(this.id.host, 12, 4)}:${this.id.port}. Sleeping for 1s.`);

                try {
                    await this.start({port: 9000, host: "127.0.0.1"});

                    console.log(`Successfully connected to ${ip.toString(this.id.host, 12, 4)}:${this.id.port}.`);
                } catch (err) {
                    setTimeout(reconnect, 1000);
                }
            };

            setTimeout(reconnect, 1000);
        });
    }

    _data({seq, frame}) {
        const opcode = frame.readUInt8();
        frame = frame.slice(1);

        switch (opcode) {
            case OPCODE_REQUEST:
                const packet = RequestPacket.decode(frame);

                const service = packet.services.find(service => this.listeners.listenerCount(service) > 0);
                if (!service) {
                    this.conn.reply(seq, new ResponsePacket().encode());
                    return;
                }

                this.listeners.emit(service, {seq, data: packet.data});
                // emit out, catch, grab result from return statement and
                break;
            default:
                throw new Error(`Unknown opcode ${opcode}.`);
        }
    }

    async handshake() {
        let packet = new HandshakePacket(
            {id: this.self, services: [...this.listeners.eventNames()]}
        ).sign(this.keys.secretKey);

        packet = HandshakePacket.decode(await this.request(OPCODE_HANDSHAKE, packet.encode()));

        this.id = packet.id;
        this.services = packet.services;
    }

    _prepare(opcode, data) {
        const header = Buffer.alloc(1);
        header.writeUInt8(opcode);

        return Buffer.concat([header, data]);
    }

    async request(opcode, data, timeout = 3000) {
        return await this.conn.request(this._prepare(opcode, data), timeout)
    }

    send(opcode, data) {
        return this.conn.send(this._prepare(opcode, data));
    }
}


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

    backend.register("all_todos", data => "hello world!")

    await backend.start({port: 9000, host: "127.0.0.1"});


    console.log(`Successfully connected to ${ip.toString(backend.id.host, 12, 4)}:${backend.id.port}.`);
}

main().catch(err => console.error(err));
