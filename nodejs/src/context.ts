import {Duplex, finished} from "stream";
import {MonteSocket} from "./monte";
import {DataPacket, Opcode, ServiceResponsePacket} from "./packet";

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