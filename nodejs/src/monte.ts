import {Duplex, DuplexOptions} from "stream";
import {EventEmitter} from "events";
import * as net from "net";
import * as crypto from "crypto";
// @ts-ignore
import blake2b from "blake2b";

import nacl from "tweetnacl";
import assert from "assert";

type MonteSocketConnectOpts = net.SocketConnectOpts & { timeout?: number };

interface MonteSocketProps extends DuplexOptions {
    sock: net.Socket;
    secret: Uint8Array;
}

enum MonteSocketReadState {
    Length,
    Body,
}

export class MonteSocket extends Duplex {
    #state: MonteSocketReadState = MonteSocketReadState.Length;
    #length: number = 0;

    #pending: EventEmitter = new EventEmitter();
    #counter: number = 0;

    #readNonce: bigint = 0n;
    #writeNonce: bigint = 0n;

    #sock: net.Socket;
    #secret: Uint8Array;

    private constructor(props: MonteSocketProps) {
        super({...props, objectMode: true});

        this.#sock = props.sock;
        this.#secret = props.secret;

        this.#sock.on('close', had_error => this.emit('close', had_error));
        this.#sock.on('connect', () => this.emit('connect'));
        this.#sock.on('drain', () => this.emit('drain'));
        this.#sock.on('end', () => this.emit('end'));
        this.#sock.on('lookup', (err, address, family, host) => {
            return this.emit('lookup', err, address, family, host);
        });
        this.#sock.on('readable', this._readable.bind(this));
        this.#sock.on('ready', () => this.emit('ready'));
        this.#sock.on('timeout', () => this.emit('timeout'));
    }

    public static async connect(opts: MonteSocketConnectOpts): Promise<MonteSocket> {
        const sock = new net.Socket();

        await new Promise((resolve, reject) => {
            sock.once('connect', resolve);
            sock.once('error', reject);
            sock.connect(opts);
        });

        sock.removeAllListeners();

        const ephemeral = nacl.box.keyPair();
        sock.write(ephemeral.publicKey);

        const publicKey = await new Promise((resolve: (value?: (PromiseLike<Uint8Array> | Uint8Array)) => void, reject) => {
            const timeout = opts.timeout && opts.timeout >= 0 && opts.timeout || 3000;

            const doTimeout = () => {
                const err = new Error(`Timed out after ${timeout} millisecond(s) trying to read peers ephemeral public key.`);
                sock.destroy(err);
                reject(err);
            };

            const handle = timeout > 0 && setTimeout(doTimeout, timeout);

            const read = () => {
                const buf = sock.read(nacl.box.publicKeyLength);
                if (!buf) sock.once('readable', read);
                else {
                    if (handle) clearTimeout(handle);
                    resolve(buf);
                }
            };

            sock.once('readable', read);
        });

        sock.removeAllListeners();

        const result = nacl.scalarMult(ephemeral.secretKey, publicKey);
        const secret = Buffer.from(blake2b(32).update(result).digest());

        return new MonteSocket({sock, secret});
    }

    public encrypt(buf: Buffer): Buffer {
        const nonce = Buffer.alloc(12);
        nonce.writeBigUInt64BE(this.#writeNonce);
        this.#writeNonce += 1n;

        const cipher = crypto.createCipheriv('aes-256-gcm', this.#secret, nonce, {authTagLength: 16});
        const ciphered = cipher.update(buf);
        cipher.final();

        return Buffer.concat([ciphered, cipher.getAuthTag()]);
    }

    public decrypt(buf: Buffer): Buffer {
        assert(buf.byteLength >= 16, 'Missing authentication tag.');

        const nonce = Buffer.alloc(12);
        nonce.writeBigUInt64BE(this.#readNonce);
        this.#readNonce += 1n;

        const decipher = crypto.createDecipheriv('aes-256-gcm', this.#secret, nonce, {authTagLength: 16});
        decipher.setAuthTag(buf.slice(buf.byteLength - 16, buf.byteLength));

        const deciphered = decipher.update(buf.slice(0, buf.byteLength - 16));
        decipher.final();

        return deciphered;
    }

    public send(seq: number, buf: Buffer): boolean {
        const header = Buffer.alloc(4);
        header.writeUInt32BE(seq);
        return this.write(Buffer.concat([header, buf]));
    }

    public async request(req: Buffer, timeout: number = 3000): Promise<Buffer> {
        const seq = this.#counter === 0 ? 1 : this.#counter += 2;
        if (this.#counter === 2 ** 32) this.#counter = 0;

        const evt = `${seq}`;

        const res = new Promise((resolve: (value?: (PromiseLike<Buffer> | Buffer)) => void, reject) => {
            const doTimeout = () => {
                this.#pending.removeAllListeners(evt);
                reject(new Error(`Timed out after ${timeout} millisecond(s) waiting for response.`))
            }

            const handle = timeout > 0 && setTimeout(doTimeout, timeout);

            this.#pending.once(evt, data => {
                if (handle) clearTimeout(handle);
                resolve(data);
            });
        });

        if (!this.send(seq, req)) {
            this.#pending.removeAllListeners(evt);
            throw new Error('Failed to write request.');
        }

        return await res;
    }

    private _readable(): void {
        while (true) {
            switch (this.#state) {
                case MonteSocketReadState.Length:
                    const header: Buffer = this.#sock.read(4);
                    if (!header) return;

                    this.#length = header.readUInt32BE();
                    this.#state = MonteSocketReadState.Body;
                    break;
                case MonteSocketReadState.Body:
                    let body: Buffer = this.#sock.read(this.#length);
                    if (!body) return;

                    body = this.decrypt(body);
                    if (body.byteLength < 4) {
                        const err = new Error(`Packet is too small: no sequence number attached.`);
                        this.#sock.destroy(err);
                        return;
                    }

                    const seq = body.readUInt32BE();
                    body = body.slice(4);

                    const evt = `${seq}`;

                    if (seq === 0 || this.#pending.listenerCount(evt) === 0) {
                        try {
                            this.push({sock: this, seq, body});
                        } catch (err) {
                            this.emit('error', err);
                        }
                    } else {
                        this.#pending.emit(evt, body);
                    }
                    this.#state = MonteSocketReadState.Length;
                    break;
            }
        }
    }

    _read(size: number) {
        setImmediate(this._readable.bind(this));
    }

    _write(chunk: any, encoding: BufferEncoding, callback: (error?: (Error | null)) => void) {
        chunk = this.encrypt(Buffer.from(chunk));

        const header = Buffer.alloc(4);
        header.writeUInt32BE(chunk.byteLength);

        this.#sock.write(Buffer.concat([header, chunk]), callback);
    }

    _final(callback: (error?: (Error | null)) => void) {
        this.#sock.end(callback);
    }
}