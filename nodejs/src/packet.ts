import nacl from "tweetnacl";
import assert from "assert";
import {ID} from "./kademlia";

export enum Opcode {
    Handshake,
    ServiceRequest,
    ServiceResponse,
    Data,
    FindNodeRequest,
    FindNodeResponse,
}

export interface Packet {
    encode(): Buffer;
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

export class ServiceRequestPacket implements Packet {
    id: number;
    services: string[] = [];
    headers: { [key: string]: string };

    public constructor(id: number, services: string[], headers: { [key: string]: string }) {
        this.id = id;
        this.services = services;
        this.headers = headers;
    }

    public encode(): Buffer {
        const id = Buffer.alloc(4);
        id.writeUInt32BE(this.id);

        const services = this.services.reduce(
            (result, service) => Buffer.concat(
                [result, Buffer.of(service.length), Buffer.from(service, "utf8")],
            ),
            Buffer.of(this.services.length),
        );

        const headersLen = Buffer.alloc(2);
        headersLen.writeUInt16BE(Object.keys(this.headers).length);

        const headers = Object.keys(this.headers).reduce(
            (result, key) => {
                const value = this.headers[key];

                const keyBuf = Buffer.concat([Buffer.of(key.length), Buffer.from(key, "utf8")]);
                const valueBuf = Buffer.concat([Buffer.alloc(2), Buffer.from(value, "utf8")]);
                valueBuf.writeUInt16BE(value.length);

                return Buffer.concat([result, keyBuf, valueBuf]);
            },
            headersLen
        );

        return Buffer.concat([id, services, headers]);
    }

    public static decode(buf: Buffer): [ServiceRequestPacket, Buffer] {
        const id = buf.readUInt32BE();
        buf = buf.slice(4);

        const servicesLen = buf.readUInt8();
        buf = buf.slice(1);

        const services: string[] = [...Array(servicesLen)].map(() => {
            const serviceLen = buf.readUInt8();
            buf = buf.slice(1);

            const service = buf.slice(0, serviceLen);
            buf = buf.slice(serviceLen);

            return service.toString("utf8");
        });

        const headersLen = buf.readUInt16BE();
        buf = buf.slice(2);

        const headers = [...Array(headersLen)].reduce((map, _) => {
            const keyLen = buf.readUInt8();
            buf = buf.slice(1);

            const key = buf.slice(0, keyLen).toString("utf8");
            buf = buf.slice(keyLen);

            const valueLen = buf.readUInt16BE();
            buf = buf.slice(2);

            const value = buf.slice(0, valueLen).toString("utf8");
            buf = buf.slice(valueLen);

            map[key] = value;
            return map;
        }, {});

        return [new ServiceRequestPacket(id, services, headers), buf];
    }
}

export class ServiceResponsePacket implements Packet {
    id: number;
    handled: boolean = false;
    headers: { [key: string]: string };

    public constructor(id: number, handled: boolean, headers: { [key: string]: string }) {
        this.id = id;
        this.handled = handled;
        this.headers = headers;
    }

    public encode(): Buffer {
        const id = Buffer.alloc(4);
        id.writeUInt32BE(this.id);

        const handled = Buffer.of(this.handled ? 1 : 0);

        const headersLen = Buffer.alloc(2);
        headersLen.writeUInt16BE(Object.keys(this.headers).length);

        const headers = Object.keys(this.headers).reduce(
            (result, key) => {
                const value = this.headers[key];

                const keyBuf = Buffer.concat([Buffer.of(key.length), Buffer.from(key, "utf8")]);
                const valueBuf = Buffer.concat([Buffer.alloc(2), Buffer.from(value, "utf8")]);
                valueBuf.writeUInt16BE(value.length);

                return Buffer.concat([result, keyBuf, valueBuf]);
            },
            headersLen
        );

        return Buffer.concat([id, handled, headers]);
    }

    public static decode(buf: Buffer): [ServiceResponsePacket, Buffer] {
        const id = buf.readUInt32BE();
        buf = buf.slice(4);

        const handled = buf.readUInt8() === 1
        buf = buf.slice(1);

        const headersLen = buf.readUInt16BE();
        buf = buf.slice(2);

        const headers = [...Array(headersLen)].reduce((map, _) => {
            const keyLen = buf.readUInt8();
            buf = buf.slice(1);

            const key = buf.slice(0, keyLen).toString("utf8");
            buf = buf.slice(keyLen);

            const valueLen = buf.readUInt16BE();
            buf = buf.slice(2);

            const value = buf.slice(0, valueLen).toString("utf8");
            buf = buf.slice(valueLen);

            map[key] = value;
            return map;
        }, {});

        return [new ServiceResponsePacket(id, handled, headers), buf];
    }
}

export class DataPacket implements Packet {
    id: number;
    data: Buffer;

    public constructor(id: number, data: Buffer) {
        this.id = id;
        this.data = data;
    }

    public encode(): Buffer {
        const id = Buffer.alloc(4);
        id.writeUInt32BE(this.id);

        const dataLen = Buffer.alloc(2);
        dataLen.writeUInt16BE(this.data.byteLength);

        return Buffer.concat([id, dataLen, this.data]);
    }

    public static decode(buf: Buffer): [DataPacket, Buffer] {
        const id = buf.readUInt32BE();
        buf = buf.slice(4);

        const dataLen = buf.readUInt16BE();
        buf = buf.slice(2);

        const data = buf.slice(0, dataLen);
        buf = buf.slice(dataLen);

        return [new DataPacket(id, data), buf];
    }
}

export class FindNodeRequest implements Packet {
    target: Uint8Array

    public constructor(target: Uint8Array) {
        this.target = target;
    }

    public encode(): Buffer {
        return Buffer.from(this.target);
    }

    public static decode(buf: Buffer): [FindNodeRequest, Buffer] {
        const target = buf.slice(0, nacl.sign.publicKeyLength);
        buf = buf.slice(nacl.sign.publicKeyLength)
        return [new FindNodeRequest(target), buf]
    }
}

export class FindNodeResponse implements Packet {
    closest: ID[]

    public constructor(closest: ID[]) {
        this.closest = closest;
    }

    public encode(): Buffer {
        return Buffer.concat([Buffer.of(this.closest.length), ...this.closest.map(id => id.encode())]);
    }

    public decode(buf: Buffer): [FindNodeResponse, Buffer] {
        const closestLen = buf.readUInt8();
        buf = buf.slice(1);

        const closest = [...Array(closestLen)].map(() => {
            const [id, leftover] = ID.decode(buf);
            buf = leftover;
            return id;
        });

        return [new FindNodeResponse(closest), buf];
    }
}