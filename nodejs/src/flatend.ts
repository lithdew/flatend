import "core-js/shim";
import {DataPacket, HandshakePacket, ID, Opcode, ServiceRequestPacket, ServiceResponsePacket} from "./packet";
import nacl from "tweetnacl";
import assert from "assert";
import {MonteSocket} from "./monte";
import * as dns from "dns";
import * as net from "net";
import * as ip from "ip";
import {promisify} from "util";
import {Duplex, finished} from "stream";


interface NodeIdentityOpts {
    keys: nacl.SignKeyPair;
    id: ID;
}

const identityOpts = (opts: any): opts is NodeIdentityOpts => opts && ("keys" in opts && "id" in opts);

type NodeOpts = NodeIdentityOpts;

const lookup = async (hostname: string) => net.isIP(hostname) ? hostname : (await promisify(dns.lookup)(hostname)).address;

export class Context extends Duplex {
    public headers: { [key: string]: string }

    _id: number;
    _sock: MonteSocket;
    _written: boolean = false;
    _headers: { [key: string]: string } = {};

    constructor(id: number, sock: MonteSocket, headers: { [key: string]: string }) {
        super({allowHalfOpen: true});

        this.headers = headers;

        this._id = id;
        this._sock = sock;

        finished(this, {readable: false}, () => {
            this._writeHeader();
            const packet = new DataPacket(this._id, Buffer.of());
            this._sock.send(0, Buffer.concat([Buffer.of(Opcode.Data), packet.encode()]));
        });
    }

    header(key: string, val: string): Context {
        this._headers[key] = val;
        return this;
    }

    send(data: string | Buffer | Uint8Array) {
        this.write(data);
        this.end();
    }

    json(data: any) {
        this.header('content-type', 'application/json');
        this.send(JSON.stringify(data));
    }

    _writeHeader() {
        if (this._written) return;

        this._written = true;
        const response = new ServiceResponsePacket(this._id, true, this._headers);
        this._sock.send(0, Buffer.concat([Buffer.of(Opcode.ServiceResponse), response.encode()]));
    }

    _write(chunk: any, encoding: BufferEncoding, callback: (error?: (Error | null)) => void) {
        if (chunk.length === 0) return;

        this._writeHeader();

        for (let i = 0; i < chunk.byteLength; i += 2048) {
            let start = i;

            let end = i + 2048;
            if (end > chunk.byteLength) end = chunk.byteLength;

            const packet = new DataPacket(this._id, chunk.slice(start, end));
            this._sock.send(0, Buffer.concat([Buffer.of(Opcode.Data), packet.encode()]));
        }

        callback();
    }

    // @ts-ignore
    async body(opts: { limit: number } = {limit: 65536}): Promise<Buffer> {
        return await (new Promise((resolve, reject) => {
            let buffer: Buffer[] = [];
            let received: number = 0;
            let complete: boolean = false;

            const done = (err?: Error) => {
                if (complete) return;

                complete = true;
                if (!err) {
                    resolve(Buffer.concat(buffer));
                } else {
                    reject(err);
                }
            }

            const onData = (data: Buffer) => {
                if (complete) return;
                received += data.byteLength;
                if (received > opts.limit) {
                    done(new Error(`request entity too large`));
                    return;
                }
                buffer.push(data);
            };

            const onEnd = (err: Error) => done(err);

            const onClose = () => {
                this.removeListener('data', onData);
                this.removeListener('error', onEnd);
                this.removeListener('end', onEnd);
                this.removeListener('close', onClose);
            }

            this.on('data', onData);
            this.on('error', onEnd);
            this.on('end', onEnd);
            this.on('close', onClose);
        }));
    }

    _read(size: number) {
        return;
    }
}

class Client {
    sock: MonteSocket;

    count: number; // streams count
    streams: Map<number, Context>; // stream id -> stream

    id?: ID | null;
    services?: string[];


    constructor(sock: MonteSocket) {
        this.sock = sock;

        this.count = 0;
        this.streams = new Map<number, Context>();
    }
}

type Handler = (ctx: Context) => void;

export class Node {
    #services: Map<string, WeakSet<Client>> = new Map<string, WeakSet<Client>>();
    #providers: WeakMap<MonteSocket, Client> = new WeakMap<MonteSocket, Client>();
    #clients: Map<string, Client> = new Map<string, Client>();
    #handlers: Map<string, Handler> = new Map<string, Handler>();

    readonly #id: ID | null = null;
    readonly #keys: nacl.SignKeyPair | null = null;

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
        return [...this.#handlers.keys()];
    }

    public register(service: string, handler: Handler) {
        assert(!this.#handlers.has(service));
        this.#handlers.set(service, handler);
    }

    public async dial(addr: string) {
        const fields = addr.trim().split(":");
        assert(fields.length === 2);

        const host = await lookup(fields[0]);

        const port = parseInt(fields[1]);
        assert(port > 0 && port < 65536)

        addr = host + ":" + port;

        let client = this.#clients.get(addr);
        if (!client) {
            client = new Client(await MonteSocket.connect({host: host, port: port}));

            client.sock.once('end', () => {
                this.#clients.delete(addr);

                const reconnect = async () => {
                    console.log(`Trying to reconnect to ${addr}. Sleeping for 1s.`);

                    try {
                        await this.dial(addr);
                    } catch (err) {
                        setTimeout(reconnect, 1000);
                    }
                };

                setTimeout(reconnect, 1000);
            });

            client.sock.on('data', this._data.bind(this));
            client.sock.on('error', console.error);

            this.#clients.set(addr, client);
            this.#providers.set(client.sock, client);
        }

        let packet = new HandshakePacket(null, this.services, null);
        if (!this.anonymous) {
            packet.id = this.#id;
            packet.signature = Buffer.from(nacl.sign(packet.payload, this.#keys!.secretKey));
        }

        const res = await client.sock.request(Buffer.concat([Buffer.of(Opcode.Handshake), packet.encode()]));
        packet = HandshakePacket.decode(res)[0];

        if (packet.id && packet.signature) {
            assert(typeof packet.id.host === "string" && ip.isEqual(packet.id.host, host) && packet.id.port === port);
            assert(nacl.sign.detached.verify(packet.payload, packet.signature, packet.id.publicKey));

            client.id = packet.id;
        }

        client.services = packet.services;

        packet.services.forEach(service => {
            if (!this.#services.has(service)) this.#services.set(service, new WeakSet<Client>());
            this.#services.get(service)!.add(client!);
        });

        console.log(`Successfully dialed ${addr}. Services: [${packet.services.join(', ')}]`);
    }

    private _data({sock, seq, body}: { sock: MonteSocket, seq: number, body: Buffer }) {
        const opcode: Opcode = body.readUInt8();
        body = body.slice(1);

        switch (opcode) {
            case Opcode.ServiceResponse:
                break;
            case Opcode.Handshake: {
                const packet = HandshakePacket.decode(body)[0];
                if (packet.id && packet.signature) {
                    assert(nacl.sign.detached.verify(packet.payload, packet.signature, packet.id.publicKey));
                }
                return;
            }
            case Opcode.ServiceRequest: {
                const packet = ServiceRequestPacket.decode(body)[0];

                const provider = this.#providers.get(sock);

                assert(provider, 'Socket is not registered as a provider.');
                assert(!provider.streams.has(packet.id), `Stream with ID ${packet.id} already exists.`);

                const service = packet.services.find(service => this.#handlers.has(service));
                if (!service) {
                    const response = new ServiceResponsePacket(packet.id, false, {});
                    sock.send(0, Buffer.concat([Buffer.of(Opcode.ServiceResponse), response.encode()]));
                    return;
                }

                const ctx = new Context(packet.id, sock, packet.headers);
                provider.streams.set(packet.id, ctx);

                const handler = this.#handlers.get(service)!;

                (async () => {
                    try {
                        await handler(ctx);
                    } catch (err) {
                        ctx.json({error: err.message});
                    }
                })()

                return;
            }
            case Opcode.Data: {
                const packet = DataPacket.decode(body)[0];

                const provider = this.#providers.get(sock);
                assert(provider, 'Socket is not registered as a provider.');

                const ctx = provider.streams.get(packet.id);
                assert(ctx, `Got data packet with stream ID ${packet.id}, but stream does not exist.`);

                if (packet.data.byteLength > 0) {
                    ctx.push(packet.data);
                    return;
                }

                provider.streams.delete(packet.id);
                ctx.push(null);

                return;
            }
            default: {
                throw new Error(`Unknown opcode ${opcode}.`);
            }
        }
    }
}