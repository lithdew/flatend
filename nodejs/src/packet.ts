import nacl from "tweetnacl";
import ip from "ip";
import assert from "assert";

const str = (s: any): s is string => typeof s === "string";

export enum Opcode {
    Handshake,
    Request,
}

export interface Packet {
    encode(): Buffer;
}

export class ID implements Packet {
    publicKey: Uint8Array = Buffer.alloc(nacl.sign.publicKeyLength);
    host: string | Buffer = "";
    port: number = 0;

    constructor(publicKey: Uint8Array, host: string | Buffer, port: number) {
        this.publicKey = publicKey;
        this.host = host;
        this.port = port;
    }

    public encode(): Buffer {
        let host = str(this.host) ? ip.toBuffer(this.host) : this.host;
        host = Buffer.concat([Buffer.of(host.byteLength === 4 ? 0 : 1), host]);

        const port = Buffer.alloc(2);
        port.writeUInt16BE(this.port);

        return Buffer.concat([this.publicKey, host, port]);
    }

    public static decode(buf: Buffer): [ID, Buffer] {
        const publicKey = buf.slice(0, nacl.sign.publicKeyLength);
        buf = buf.slice(nacl.sign.publicKeyLength);

        const hostHeader = buf.readUInt8();
        buf = buf.slice(1);

        assert(hostHeader === 0 || hostHeader === 1);

        const hostLen = hostHeader === 0 ? 4 : 16;
        const host = ip.toString(buf.slice(0, hostLen), 0, hostLen);
        buf = buf.slice(hostLen);

        const port = buf.readUInt16BE();
        buf = buf.slice(2);

        return [new ID(publicKey, host, port), buf];
    }
}

export class HandshakePacket implements Packet {
    id: ID | null = null;
    services: string[] = [];
    signature: Buffer | null = null;

    constructor(id: ID | null, services: string[], signature: Buffer | null) {
        this.id = id;
        this.services = services;
        this.signature = signature;
    }

    get payload() {
        return Buffer.concat(
            [
                this.id!.encode(),
                ...this.services.map(service => Buffer.from(service, "utf8")),
            ]
        );
    }

    public encode(): Buffer {
        const id = this.id ? Buffer.concat([Buffer.of(1), this.id.encode()]) : Buffer.of(0);
        const services = this.services.reduce(
            (result, service) => Buffer.concat(
                [result, Buffer.of(service.length), Buffer.from(service, "utf8")],
            ),
            Buffer.of(this.services.length),
        );
        const signature = this.id && this.signature ? this.signature : Buffer.of();

        return Buffer.concat([id, services, signature]);
    }

    public static decode(buf: Buffer): [HandshakePacket, Buffer] {
        const header = buf.readUInt8();
        buf = buf.slice(1);

        assert(header === 0 || header === 1);

        const result: [ID | null, Buffer] = header === 1 ? ID.decode(buf) : [null, buf];

        const id = result[0];
        buf = result[1];

        const size = buf.readUInt8();
        buf = buf.slice(1);

        const services: string[] = [...Array(size)].map(() => {
            const size = buf.readUInt8();
            buf = buf.slice(1);

            const service = buf.slice(0, size);
            buf = buf.slice(size);

            return service.toString("utf8");
        });

        let signature: Buffer | null = null;
        if (id) {
            signature = buf.slice(0, nacl.sign.signatureLength);
            buf = buf.slice(nacl.sign.signatureLength);
        }

        return [new HandshakePacket(id, services, signature), buf];
    }
}

export class RequestPacket implements Packet {
    services: string[] = [];
    data: Buffer = Buffer.of();

    constructor(services: string[], data: Buffer) {
        this.services = services;
        this.data = data;
    }

    public encode(): Buffer {
        const services = this.services.reduce(
            (result, service) => Buffer.concat(
                [result, Buffer.of(service.length), Buffer.from(service, "utf8")],
            ),
            Buffer.of(),
        );

        const data = Buffer.concat([Buffer.alloc(4), this.data]);
        data.writeUInt32BE(this.data.byteLength);

        return Buffer.concat([services, data]);
    }

    public static decode(buf: Buffer): [RequestPacket, Buffer] {
        const serviceHeader = buf.readUInt8();
        buf = buf.slice(1);

        const services: string[] = [...Array(serviceHeader)].map(() => {
            const size = buf.readUInt8();
            buf = buf.slice(1);

            const service = buf.slice(0, size);
            buf = buf.slice(size);

            return service.toString("utf8");
        });

        const dataHeader = buf.readUInt32BE();
        buf = buf.slice(4);

        const data = buf.slice(0, dataHeader);
        buf = buf.slice(dataHeader);

        return [new RequestPacket(services, data), buf];
    }
}

export class ResponsePacket implements Packet {
    data: Buffer | null = null;

    constructor(data: Buffer | null) {
        this.data = data;
    }

    public encode(): Buffer {
        if (this.data) {
            const data = Buffer.concat([Buffer.alloc(5), this.data]);
            data.writeUInt8(1);
            data.writeUInt32BE(this.data.byteLength, 1);
            return data;
        }
        return Buffer.of(0);
    }

    public static decode(buf: Buffer): [ResponsePacket, Buffer] {
        const header = buf.readUInt8();
        buf = buf.slice(1);

        assert(header === 0 || header === 1);

        if (header === 0) return [new ResponsePacket(null), buf];

        const size = buf.readUInt32BE();
        buf = buf.slice(4);

        const data = buf.slice(0, size);
        buf = buf.slice(size);

        return [new ResponsePacket(data), buf];
    }
}