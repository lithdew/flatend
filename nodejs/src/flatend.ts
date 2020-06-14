import "core-js/shim";

import {EventEmitter} from "events";
import {HandshakePacket, ID, Opcode, RequestPacket, ResponsePacket} from "./packet";
import nacl from "tweetnacl";
import assert from "assert";
import {MonteSocket} from "./monte";
import * as dns from "dns";
import * as net from "net";
import * as ip from "ip";
import {promisify} from "util";


interface NodeIdentityOpts {
    keys: nacl.SignKeyPair;
    id: ID;
}

const identityOpts = (opts: any): opts is NodeIdentityOpts => opts && ("keys" in opts && "id" in opts);
const str = (p: any): p is string => typeof p === "string";

type NodeOpts = NodeIdentityOpts;

const lookup = async (hostname: string) => net.isIP(hostname) ? hostname : (await promisify(dns.lookup)(hostname)).address;

interface Peer {
    sock: MonteSocket;
    id?: ID | null;
    services?: string[];
}

export class Node {
    #peers: Map<string, Peer> = new Map<string, Peer>();
    #listeners: EventEmitter = new EventEmitter();

    #id: ID | null = null;
    #keys: nacl.SignKeyPair | null = null;

    public constructor(opts?: NodeOpts) {
        if (identityOpts(opts)) {
            this.#id = opts.id;
            this.#keys = opts.keys;
        }
    }

    get anonymous(): boolean {
        return this.#id === null && this.#keys === null
    };

    get services(): string[] {
        return this.#listeners.eventNames().map(name => name.toString())
    }

    public register(service: string, handler: (data: Buffer) => any) {
        assert(this.#listeners.listenerCount(service) === 0);

        this.#listeners.on(service, async ({sock, seq, data}) => {
            let response: Buffer | null;
            try {
                let result = await handler(data);
                if (result && !str(result) && !Buffer.isBuffer(result)) result = JSON.stringify(result);
                if (result && str(result)) result = Buffer.from(result, "utf8");
                response = result;
            } catch (err) {
                response = Buffer.from(JSON.stringify({error: err}), "utf8");
            }
            sock.send(seq, new ResponsePacket(response).encode());
        });
    }

    public async dial(addr: string) {
        const fields = addr.trim().split(":");
        assert(fields.length === 2);

        const host = await lookup(fields[0]);

        const port = parseInt(fields[1]);
        assert(port > 0 && port < 65536)

        addr = host + ":" + port;

        let peer = this.#peers.get(addr);
        if (!peer) {
            peer = {sock: await MonteSocket.connect({host: host, port: port})};

            peer.sock.once('end', () => {
                this.#peers.delete(addr);

                const reconnect = async () => {
                    console.log(`Trying to reconnect to ${addr}. Sleeping for 1s.`);

                    try {
                        await this.dial(addr);
                        console.log(`Successfully connected to ${addr}.`);
                    } catch (err) {
                        setTimeout(reconnect, 1000);
                    }
                };

                setTimeout(reconnect, 1000);
            });

            peer.sock.on('data', this._data.bind(this));
            peer.sock.on('error', console.error);

            this.#peers.set(addr, peer);
        }

        let packet = new HandshakePacket(null, this.services, null);
        if (!this.anonymous) {
            packet.id = this.#id;
            packet.signature = Buffer.from(nacl.sign(packet.payload, this.#keys!.secretKey));
        }

        const res = await peer.sock.request(Buffer.concat([Buffer.of(Opcode.Handshake), packet.encode()]));
        packet = HandshakePacket.decode(res)[0];

        if (packet.id && packet.signature) {
            assert(typeof packet.id.host === "string" && ip.isEqual(packet.id.host, host) && packet.id.port === port);
            assert(nacl.sign.detached.verify(packet.payload, packet.signature, packet.id.publicKey));

            peer.id = packet.id;
        }

        peer.services = packet.services;

        // TODO(kenta): register services to node
    }

    private _data({sock, seq, body}: { sock: MonteSocket, seq: number, body: Buffer }) {
        const opcode: Opcode = body.readUInt8();
        body = body.slice(1);

        switch (opcode) {
            case Opcode.Handshake: {
                const packet = HandshakePacket.decode(body)[0];
                if (packet.id && packet.signature) {
                    assert(nacl.sign.detached.verify(packet.payload, packet.signature, packet.id.publicKey));
                }
                return;
            }
            case Opcode.Request: {
                const packet = RequestPacket.decode(body)[0];

                const service = packet.services.find(service => this.#listeners.listenerCount(service) > 0);
                if (!service) {
                    sock.send(seq, new ResponsePacket(null).encode());
                    return;
                }

                this.#listeners.emit(service, {sock, seq, data: packet.data});

                return;
            }
            default: {
                throw new Error(`Unknown opcode ${opcode}.`);
            }
        }
    }
}