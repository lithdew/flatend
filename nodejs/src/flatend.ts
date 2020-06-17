import "core-js";

import {
    DataPacket,
    FindNodeRequest,
    FindNodeResponse,
    HandshakePacket,
    ID,
    Opcode,
    ServiceRequestPacket,
    ServiceResponsePacket,
    Table
} from "./packet";
import nacl from "tweetnacl";
import assert from "assert";
import {MonteSocket} from "./monte";
import * as dns from "dns";
import {promisify} from "util";
import {Duplex, finished} from "stream";


interface NodeIdentityOpts {
    keys: nacl.SignKeyPair;
    id: ID;
}

const identityOpts = (opts: any): opts is NodeIdentityOpts => opts && ("keys" in opts && "id" in opts);

type NodeOpts = NodeIdentityOpts;

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

class Provider {
    id?: ID;
    sock: MonteSocket;
    services: Set<string>

    count: number = 0;
    streams: Map<number, Context> = new Map<number, Context>();

    constructor(id: ID | undefined, sock: MonteSocket, services = new Set<string>()) {
        this.id = id;
        this.sock = sock;
        this.services = services;
    }

    get addr(): string {
        if (!this.id) return "???";
        return this.id.host + ":" + this.id.port;
    }
}

type Handler = (ctx: Context) => void;

const resolve = async (addr: string): Promise<[string, number]> => {
    const [host, port] = addr.trim().split(":");

    const resolvedHost = (await promisify(dns.lookup)(host)).address;

    const resolvedPort = parseInt(port);
    assert(resolvedPort >= 0 && resolvedPort < 65536);

    return [resolvedHost, resolvedPort];
}

export class Node {
    #services: Map<string, WeakSet<Provider>> = new Map<string, WeakSet<Provider>>();
    #providers: WeakMap<MonteSocket, Provider> = new WeakMap<MonteSocket, Provider>();
    #clients: Map<string, MonteSocket> = new Map<string, MonteSocket>();
    #handlers: Map<string, Handler> = new Map<string, Handler>();
    #table: Table;

    readonly #id: ID | null = null;
    readonly #keys: nacl.SignKeyPair | null = null;

    public constructor(opts?: NodeOpts) {
        if (identityOpts(opts)) {
            this.#id = opts.id;
            this.#keys = opts.keys;
        }

        this.#table = new Table(this.#id?.publicKey ?? Buffer.alloc(nacl.sign.publicKeyLength));
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
        const [host, port] = await resolve(addr);

        let sock = this.#clients.get(addr);
        if (!sock) {
            sock = await MonteSocket.connect({host, port});

            this.#clients.set(addr, sock);

            sock.on('data', this._data.bind(this));
            sock.on('error', console.error);
            sock.once('end', () => this.#clients.delete(addr));
        }

        await this.probe(sock);
    }

    private createHandshakePacket(): HandshakePacket {
        let packet = new HandshakePacket(null, this.services, null);
        if (!this.anonymous) {
            packet.id = this.#id;
            packet.signature = Buffer.from(nacl.sign(packet.payload, this.#keys!.secretKey));
        }
        return packet;
    }

    private onPeerJoin(sock: MonteSocket, packet: HandshakePacket): Provider {
        let id: ID | undefined;

        if (packet.id && packet.signature) {
            if (!nacl.sign.detached.verify(packet.payload, packet.signature, packet.id.publicKey)) {
                throw new Error(`Handshake packet signature is malformed.`);
            }
            id = packet.id;
        }

        if (id) this.#table.update(id);

        let provider = this.#providers.get(sock);
        if (!provider) {
            provider = new Provider(id, sock, new Set<string>(packet.services));
            this.#providers.set(sock, provider);

            sock.once('end', () => {
                if (id) this.#table.delete(id.publicKey);
                provider!.services.forEach(service => this.#services.get(service)?.delete(provider!));
                this.#providers.delete(sock);
            });
        }

        provider.services.forEach(service => {
            if (!this.#services.has(service)) this.#services.set(service, new WeakSet<Provider>());
            this.#services.get(service)!.add(provider!);
        });

        return provider;
    }

    private async probe(sock: MonteSocket) {
        const packet = HandshakePacket.decode(
            await sock.request(
                Buffer.concat([
                    Buffer.of(Opcode.Handshake),
                    this.createHandshakePacket().encode(),
                ]),
            ),
        )[0];

        const provider = this.onPeerJoin(sock, packet);

        console.log(`Successfully dialed ${provider.addr}. Services: [${packet.services.join(', ')}]`);

        sock.once('end', () => {
            let count = 8;

            const reconnect = async () => {
                if (count-- === 0) {
                    console.log(`Tried 8 times reconnecting to ${provider.addr}. Giving up.`);
                    return;
                }

                console.log(`Trying to reconnect to ${provider.addr}. Sleeping for 1s.`);

                try {
                    await this.dial(provider.addr!);
                } catch (err) {
                    setTimeout(reconnect, 1000);
                }
            };

            setTimeout(reconnect, 1000);
        });
    }

    private _data({sock, seq, body}: { sock: MonteSocket, seq: number, body: Buffer }) {
        const opcode: Opcode = body.readUInt8();
        body = body.slice(1);

        switch (opcode) {
            case Opcode.ServiceResponse:
                break;
            case Opcode.Handshake: {
                const packet = HandshakePacket.decode(body)[0];
                const provider = this.onPeerJoin(sock, packet);

                console.log(`${provider.addr} has connected to you. Services: [${[...provider.services].join(", ")}]`);

                sock.send(seq, this.createHandshakePacket().encode());

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
            case Opcode.FindNodeRequest: {
                const packet = FindNodeRequest.decode(body)[0];
                sock.send(seq, new FindNodeResponse(this.#table.closestTo(packet.target)).encode())
                return;
            }
            default: {
                throw new Error(`Unknown opcode ${opcode}.`);
            }
        }
    }
}