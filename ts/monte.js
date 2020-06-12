import {Duplex} from "stream";
import {EventEmitter} from "events";
import net from "net";
import nacl from "tweetnacl";
import blake2b from "blake2b";
import crypto from "crypto";

/**
 *
 * @param {Socket} sock
 * @param {number} n
 * @param {number} timeout
 * @return {Promise<Buffer>}
 */
const read = (sock, n, timeout = 3000) => new Promise((resolve, reject) => {
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

class MonteSocket extends Duplex {
    /**
     *
     * @param {module:net.Socket} conn
     * @param {Uint8Array} secret
     * @param [opts]
     */
    constructor(conn, secret, opts) {
        super({...opts, objectMode: true});

        this.conn = conn;
        this.secret = secret;

        this.readNonce = 0n;
        this.writeNonce = 0n;

        this.pending = new EventEmitter();
        this.counter = 0;

        this.conn.on('close', had_error => this.emit('close', had_error));
        this.conn.on('connect', () => this.emit('connect'));
        this.conn.on('drain', () => this.emit('drain'));
        this.conn.on('end', () => this.emit('end'));
        this.conn.on('error', err => this.emit('error', err));

        this.conn.on('lookup', (err, address, family, host) => {
            return this.emit('lookup', err, address, family, host);
        });

        this.conn.on('ready', () => this.emit('ready'));
        this.conn.on('timeout', () => this.emit('timeout'));

        this.conn.on('readable', this._readable.bind(this));
    }

    static async dial(opts) {
        const conn = new net.Socket(opts);

        await new Promise((resolve, reject) => {
            conn.once('connect', () => {
                conn.removeAllListeners();
                resolve();
            });
            conn.once('error', err => {
                conn.removeAllListeners();
                reject(err);
            });
            conn.connect(opts);
        });

        const ephemeral = nacl.box.keyPair();
        conn.write(ephemeral.publicKey);

        const peerPublicKey = await read(conn, nacl.box.publicKeyLength);

        let secret = nacl.scalarMult(ephemeral.secretKey, Buffer.from(peerPublicKey.buffer));
        secret = Buffer.from(blake2b(32).update(secret).digest());

        return new MonteSocket(conn, secret);
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

    send(buf) {
        return this._send(0, buf);
    }

    reply(seq, buf) {
        return this._send(seq, buf);
    }

    async request(req, timeout = 3000) {
        const seq = this.counter === 0 ? 1 : this.counter++;
        if (this.counter === 2 ** 32) this.counter = 0;

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

        if (!this._send(seq, Buffer.from(req))) {
            this.pending.removeAllListeners(`${seq}`);
            throw new Error("Failed to write request.");
        }

        return await response;
    }

    _send(seq, buf) {
        const header = Buffer.alloc(4);
        header.writeUInt32BE(seq);

        return this.write(Buffer.concat([header, buf]));
    }

    _write(chunk, encoding, callback) {
        chunk = this.encrypt(Buffer.from(chunk));

        const header = Buffer.alloc(4);
        header.writeUInt32BE(chunk.byteLength);

        this.conn.write(Buffer.concat([header, chunk]), callback);
    }

    _read(_size) {
        setImmediate(this._readable.bind(this));
    }

    _readable() {
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
            if (frame.byteLength <= 4) {
                this.conn.destroy(new Error(`Packet is too small: no sequence number field found in packet.`));
                return;
            }

            const seq = frame.readUInt32BE();
            frame = frame.slice(4);

            if (seq === 0 || this.pending.listenerCount(`${seq}`) === 0) {
                try {
                    this.push({seq, frame});
                } catch (err) {
                    this.emit("error", err);
                }
            } else {
                this.pending.emit(`${seq}`, frame);
            }
        }
    }
}

export default MonteSocket;