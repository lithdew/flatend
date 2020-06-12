import nacl from "tweetnacl";
import ip from "ip";

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
            (acc, service) => Buffer.concat(
                [acc, Buffer.of(service.length), Buffer.from(service, "utf8")],
            ),
            Buffer.of(this.services.length),
        );

        return Buffer.concat([
            this.id.encode(),
            services,
            this.signature,
        ]);
    }

    /**
     *
     * @param {Buffer} buf
     * @return {HandshakePacket}
     */
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
            (acc, service) => Buffer.concat(
                [acc, Buffer.of(service.length), Buffer.from(service, "utf8")],
            ),
            Buffer.of(this.services.length),
        );

        const data = Buffer.alloc(4);
        data.writeUInt32BE(this.data.byteLength);

        return Buffer.concat([services, data, this.data])
    }

    /**
     *
     * @param {Buffer} buf
     * @return {RequestPacket}
     */
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

class ResponsePacket {
    constructor(
        data = null
    ) {
        this.data = data;
    }

    encode() {
        if (this.data) {
            const header = Buffer.alloc(5);
            header.writeUInt8(1);
            header.writeUInt32BE(this.data.byteLength, 1);
            return Buffer.concat([header, this.data]);
        }
        return Buffer.of(0);
    }

    /**
     *
     * @param {Buffer} buf
     */
    static decode(buf) {
        const handled = buf.readUInt8() === 1;
        buf = buf.slice(1);

        if (!handled) return new ResponsePacket();

        const size = buf.readUInt32BE();
        buf = buf.slice(4);

        return new ResponsePacket(buf.slice(0, size));
    }
}

export {ID, HandshakePacket, RequestPacket, ResponsePacket};