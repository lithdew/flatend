import nacl from "tweetnacl";
import * as net from "net";
import blake2b from 'blake2b';
import * as crypto from "crypto";
import {EventEmitter} from "events";
import ip from "ip";

const OPCODE_HANDSHAKE = 0;
const OPCODE_REQUEST = 1;

class RequestPacket {
    constructor(
        {
            services = [],
            data = Buffer.of()
        }
    ) {
        this.services = services;
        this.data = data;
    }

    encode() {
        const services = this.services.reduce(
            (acc, service) => Buffer.concat([acc, Buffer.of(service.length), Buffer.from(service, "utf8")]),
            Buffer.of(this.services.length),
        );

        const data = Buffer.alloc(4);
        data.writeUInt32BE(this.data.byteLength);

        return Buffer.concat([services, data, this.data])
    }

    static decode(buf) {
        let size = buf.readUInt8();
        buf = buf.slice(1);

        const services = [...Array(size)].map(() => {
            const size = buf.readUInt8();
            buf = buf.slice(1);

            const service = buf.slice(0, size);
            buf = buf.slice(size);

            return service.toString("utf8");
        });

        size = buf.readUInt32BE();
        buf = buf.slice(4);

        const data = buf.slice(0, size);

        return new RequestPacket({services, data});
    }
}


class ID {
    constructor(
        {
            publicKey = Buffer.alloc(nacl.sign.publicKeyLength),
            host = "",
            port = 0,
        },
    ) {
        this.publicKey = publicKey;
        this.host = host;
        this.port = port;
    }

    /**
     *
     * @param {Buffer} buf
     * @return {[ID, Buffer]}
     */
    static decode(buf) {
        const publicKey = buf.slice(0, nacl.sign.publicKeyLength);
        buf = buf.slice(nacl.sign.publicKeyLength);

        const hostType = buf.readUInt8();
        buf = buf.slice(1);

        let host;
        switch (hostType) {
            case 0:
                [host, buf] = [buf.slice(0, 4), buf.slice(4)];
                break;
            case 1:
                [host, buf] = [buf.slice(0, 16), buf.slice(16)]
                break;
            case 2:
                [host, buf] = [Buffer.of(), buf]
                break;
            default:
                throw new Error(`Unknown host type '${hostType}' specified in handshake packet.`)
        }

        const port = buf.readUInt16BE();
        buf = buf.slice(2)

        return [new ID({publicKey, host, port}), buf];
    }

    /**
     *
     * @return {Buffer}
     */
    encode() {
        let host;
        if (typeof this.host === "string") {
            if (ip.isV4Format(this.host)) {
                host = Buffer.concat([Buffer.of(0), ip.toBuffer(this.host)]);
            } else if (ip.isV6Format(this.host)) {
                host = Buffer.concat([Buffer.of(1), ip.toBuffer(this.host)]);
            } else {
                host = Buffer.of(2);
            }
        } else if (Buffer.isBuffer(this.host)) {
            const type = this.host.byteLength === 4 ? 0 : this.host.byteLength === 16 ? 1 : 2;
            host = Buffer.concat([Buffer.of(type), this.host]);
        }

        const port = Buffer.alloc(2);
        port.writeUInt16BE(this.port);

        return Buffer.concat([this.publicKey, host, port]);
    }
}

class HandshakePacket {
    constructor({id = new ID({}), services = [], signature = Buffer.alloc(nacl.sign.signatureLength)}) {
        this.id = id;
        this.services = services;
        this.signature = signature;
    }

    get payload() {
        return Buffer.concat([
            this.id.encode(),
            ...this.services.map(service => Buffer.from(service, "utf8")),
        ])
    }

    sign(secretKey) {
        this.signature = Buffer.from(nacl.sign.detached(this.payload, secretKey));
        return this;
    }

    encode() {
        const services = this.services.reduce(
            (acc, service) => Buffer.concat([acc, Buffer.of(service.length), Buffer.from(service, "utf8")]),
            Buffer.of(this.services.length),
        );

        return Buffer.concat([
            this.id.encode(),
            services,
            this.signature,
        ]);
    }

    static decode(buf) {
        let decoded = ID.decode(buf);

        let id = decoded[0];
        buf = decoded[1];

        const size = buf.readUInt8();
        buf = buf.slice(1);

        const services = [...Array(size)].map(() => {
            const size = buf.readUInt8();
            buf = buf.slice(1);

            const service = buf.slice(0, size);
            buf = buf.slice(size);

            return service.toString("utf8");
        });

        const signature = buf.slice(0, nacl.sign.signatureLength);
        buf = buf.slice(nacl.sign.signatureLength);

        const packet = new HandshakePacket({id, services, signature});

        if (!nacl.sign.detached.verify(packet.payload, packet.signature, packet.id.publicKey)) {
            throw new Error(`Signature specified in handshake packet is invalid.`)
        }

        return packet;
    }
}

const sized = data => {
    const buf = Buffer.alloc(4);
    buf.writeUInt32BE(data.byteLength);
    return Buffer.concat([buf, data]);
}

/**
 *
 * @param {Socket} sock
 * @param {number} n
 * @param {number} timeout
 * @return {Promise<Buffer>}
 */
const readExactly = (sock, n, timeout = 3000) => new Promise((resolve, reject) => {
    const doTimeout = () => {
        const err = new Error(`Timed out after ${timeout} millisecond(s) trying to read ${n} byte(s).`);
        sock.destroy(err);
        reject(err);
    }

    const handle = timeout > 0 && setTimeout(doTimeout, timeout);

    const tryRead = () => {
        const buf = sock.read(n);
        if (!buf) {
            sock.once('readable', tryRead);
        } else {
            if (handle) clearTimeout(handle);
            resolve(buf);
        }
    }

    sock.once('readable', tryRead);
});

class Client {
    /**
     *
     * @param {Node} node
     * @param {net.Socket} conn
     * @param {Buffer} secret
     */
    constructor(node, opts, conn, secret) {
        this.readNonce = 0n;
        this.writeNonce = 0n;

        this.pending = new EventEmitter();
        this.counter = 0;

        this.node = node;
        this.opts = opts;
        this.conn = conn;
        this.secret = secret;

        this.conn.on('readable', this._read.bind(this));
        this.conn.on('close', () => {
            this.node.clients.delete(this.opts);

            const reconnect = async () => {
                console.log(`Trying to reconnect to ${ip.toString(this.id.host, 12, 4)}:${this.id.port}. Sleeping for 1s.`);

                try {
                    await this.node.dial(this.opts);
                } catch (err) {
                    setTimeout(reconnect, 1000);
                }
            };

            setTimeout(reconnect, 1000);
        });
    }

    async handshake() {
        const packet = new HandshakePacket(
            {
                id: this.node.id,
                services: ["get_todos"]
            },
        ).sign(this.node.secretKey);

        const res = await this.request(Buffer.concat([Buffer.of(OPCODE_HANDSHAKE), packet.encode()]));
        const info = HandshakePacket.decode(res);

        this.id = info.id;
        this.services = info.services;
    }

    async request(req, timeout = 3000) {
        const seq = this.counter === 0 ? 1 : this.counter++;
        if (this.counter === 2 ** 32) this.counter = 0;

        const header = Buffer.alloc(4);
        header.writeUInt32BE(seq);

        const response = new Promise((resolve, reject) => {
            const doTimeout = () => {
                this.pending.removeAllListeners(`${seq}`);
                reject(new Error(`Timed out waiting for response to request.`));
            }

            const handle = timeout > 0 && setTimeout(doTimeout, timeout);

            this.pending.once(`${seq}`, data => {
                if (handle) clearTimeout(handle);
                resolve(data);
            });
        });

        req = Buffer.concat([header, Buffer.from(req)]);
        this.conn.write(sized(this.encrypt(req)));

        return await response;
    }

    send(buf) {
        buf = Buffer.concat([Buffer.alloc(4), Buffer.from(buf)]);
        this.conn.write(sized(this.encrypt(buf)));
    }

    reply(seq, buf) {
        const header = Buffer.alloc(4);
        header.writeUInt32BE(seq);

        this.conn.write(sized(this.encrypt(Buffer.concat([header, buf]))));
    }

    /**
     *
     * @param {crypto.BinaryLike} buf
     * @returns {Buffer}
     */
    encrypt(buf) {
        const nonce = Buffer.alloc(12);
        nonce.writeBigUInt64BE(this.writeNonce);
        this.writeNonce++

        const cipher = crypto.createCipheriv("aes-256-gcm", this.secret, nonce, {authTagLength: 16});
        const ciphered = cipher.update(buf);
        cipher.final();

        return Buffer.concat([ciphered, cipher.getAuthTag()]);
    }

    /**
     *
     * @param {crypto.BinaryLike} buf
     * @returns {Buffer}
     */
    decrypt(buf) {
        if (buf.byteLength < 16) {
            throw new Error("Data to be decrypted must be at least 16 bytes.");
        }

        const nonce = Buffer.alloc(12);
        nonce.writeBigUInt64BE(this.readNonce);
        this.readNonce++

        const decipher = crypto.createDecipheriv("aes-256-gcm", this.secret, nonce, {authTagLength: 16});
        decipher.setAuthTag(buf.slice(buf.byteLength - 16, buf.byteLength));

        const deciphered = decipher.update(buf.slice(0, buf.byteLength - 16));
        decipher.final();

        return deciphered;
    }

    _read() {
        while (true) {
            const header = this.conn.read(4);
            if (!header) return;

            const length = header.readUInt32BE();

            let frame = this.conn.read(length);
            if (!frame) {
                this.conn.unshift(header);
                return;
            }

            frame = this.decrypt(frame);
            if (frame.byteLength <= 5) { // seq, opcode
                this.conn.destroy(new Error(`Unexpected EOF reading message.`));
                return;
            }

            const seq = frame.readUInt32BE();
            frame = frame.slice(4);

            if (seq === 0 || this.pending.listenerCount(`${seq}`) === 0) {
                const opcode = frame.readUInt8();
                frame = frame.slice(1);

                if (opcode !== OPCODE_REQUEST) {
                    throw new Error(`Got unexpected opcode ${opcode}.`);
                }

                const packet = RequestPacket.decode(frame);
                const body = JSON.parse(packet.data.toString("utf8"));

                // console.log(body);

                this.reply(seq, packet.data);
            } else {
                this.pending.emit(`${seq}`, frame);
            }
        }
    }
}

class Node {
    constructor({keys = nacl.sign.keyPair(), host = "", port = 0}) {
        this.clients = new Map();

        this.id = new ID({publicKey: keys.publicKey, host: host, port: port});
        this.publicKey = keys.publicKey;
        this.secretKey = keys.secretKey;
        this.host = host;
        this.port = port;
    }

    listen() {
        this.srv = new net.Server();
        this.srv.listen(this.port, this.host);

        return new Promise((resolve, reject) => {
            this.srv.on('error', reject);
            this.srv.on('listening', () => {
                const info = this.srv.address();
                switch (info.family) {
                    case "IPv6":
                        console.log(`Listening for connections on [${info.address}]:${info.port}.`);
                        break;
                    default:
                        console.log(`Listening for connections on ${info.address}:${info.port}.`);
                        break;
                }

                this.id.host = this.host = info.address;
                this.id.port = this.port = info.port;

                resolve();
            });
        });
    }

    async dial(opts) {
        if (this.clients.has(opts)) {
            return this.clients.get(opts);
        }

        const conn = new net.Socket();

        await new Promise((resolve, reject) => {
            conn.once('error', err => reject(err));
            conn.once('connect', resolve);
            conn.connect(opts);
        });

        const {publicKey, secretKey} = nacl.box.keyPair();
        conn.write(publicKey);

        const peerPublicKey = await readExactly(conn, nacl.box.publicKeyLength);

        let sessionKey = nacl.scalarMult(secretKey, Buffer.from(peerPublicKey.buffer));
        sessionKey = blake2b(32).update(sessionKey).digest();
        sessionKey = Buffer.from(sessionKey);

        const client = new Client(this, opts, conn, sessionKey);
        await client.handshake();

        this.clients.set(opts, client);

        console.log(`Successfully connected to ${ip.toString(client.id.host, 12, 4)}:${this.id.port}.`);

        return client;
    }
}

async function main() {
    const node = new Node({host: "127.0.0.1"});
    await node.listen();

    await node.dial({port: 9000, host: "127.0.0.1"});
}

main().catch(err => console.error(err));