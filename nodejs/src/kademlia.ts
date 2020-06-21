import nacl from "tweetnacl";
import ipaddr, { IPv4, IPv6 } from "ipaddr.js";
import assert from "assert";

export enum UpdateResult {
  New,
  Ok,
  Full,
  Fail,
}

const leadingZeros = (buf: Uint8Array): number => {
  const i = buf.findIndex((b) => b != 0);
  if (i === -1) return buf.byteLength * 8;

  let b = buf[i] >>> 0;
  if (b === 0) return i * 8 + 8;
  return i * 8 + ((7 - ((Math.log(b) / Math.LN2) | 0)) | 0);
};

const xor = (a: Uint8Array, b: Uint8Array): Uint8Array => {
  const c = Buffer.alloc(Math.min(a.byteLength, b.byteLength));
  for (let i = 0; i < c.byteLength; i++) c[i] = a[i] ^ b[i];
  return c;
};

export class ID {
  publicKey: Uint8Array = Buffer.alloc(nacl.sign.publicKeyLength);
  host: IPv4 | IPv6;
  port: number = 0;

  constructor(publicKey: Uint8Array, host: IPv4 | IPv6, port: number) {
    this.publicKey = publicKey;
    this.host = host;
    this.port = port;
  }

  get addr(): string {
    let host = this.host;
    if (host.kind() === "ipv6" && (<IPv6>host).isIPv4MappedAddress()) {
      host = (<IPv6>host).toIPv4Address();
    }
    return host.toString() + ":" + this.port;
  }

  public encode(): Buffer {
    let host = Buffer.of(...this.host.toByteArray());
    host = Buffer.concat([Buffer.of(host.byteLength === 4 ? 0 : 1), host]);

    const port = Buffer.alloc(2);
    port.writeUInt16BE(this.port);

    return Buffer.concat([this.publicKey, host, port]);
  }

  public static decode(buf: Buffer): [ID, Buffer] {
    const publicKey = Uint8Array.from(buf.slice(0, nacl.sign.publicKeyLength));
    buf = buf.slice(nacl.sign.publicKeyLength);

    const hostHeader = buf.readUInt8();
    buf = buf.slice(1);

    assert(hostHeader === 0 || hostHeader === 1);

    const hostLen = hostHeader === 0 ? 4 : 16;
    const host = ipaddr.fromByteArray([...buf.slice(0, hostLen)]);
    buf = buf.slice(hostLen);

    const port = buf.readUInt16BE();
    buf = buf.slice(2);

    return [new ID(publicKey, host, port), buf];
  }
}

export class Table {
  buckets: Array<Array<ID>> = [
    ...Array(nacl.sign.publicKeyLength * 8),
  ].map(() => []);

  pub: Uint8Array;
  cap: number = 16;
  length: number = 0;

  public constructor(
    pub: Uint8Array = Buffer.alloc(nacl.sign.publicKeyLength)
  ) {
    this.pub = pub;
  }

  private bucketIndex(pub: Uint8Array): number {
    if (Buffer.compare(pub, this.pub) === 0) return 0;
    return leadingZeros(xor(pub, this.pub));
  }

  public update(id: ID): UpdateResult {
    if (Buffer.compare(id.publicKey, this.pub) === 0) return UpdateResult.Fail;

    const bucket = this.buckets[this.bucketIndex(id.publicKey)];

    const i = bucket.findIndex(
      (item) => Buffer.compare(item.publicKey, id.publicKey) === 0
    );
    if (i >= 0) {
      bucket.unshift(...bucket.splice(i, 1));
      return UpdateResult.Ok;
    }

    if (bucket.length < this.cap) {
      bucket.unshift(id);
      this.length++;
      return UpdateResult.New;
    }
    return UpdateResult.Full;
  }

  public delete(pub: Uint8Array): boolean {
    const bucket = this.buckets[this.bucketIndex(pub)];
    const i = bucket.findIndex((id) => Buffer.compare(id.publicKey, pub) === 0);
    if (i >= 0) {
      bucket.splice(i, 1);
      this.length--;
      return true;
    }
    return false;
  }

  public has(pub: Uint8Array): boolean {
    const bucket = this.buckets[this.bucketIndex(pub)];
    return !!bucket.find((id) => Buffer.compare(id.publicKey, pub) === 0);
  }

  public closestTo(pub: Uint8Array, k = this.cap): ID[] {
    const closest: ID[] = [];

    const fill = (i: number) => {
      const bucket = this.buckets[i];
      for (let i = 0; closest.length < k && i < bucket.length; i++) {
        if (Buffer.compare(bucket[i].publicKey, pub) != 0)
          closest.push(bucket[i]);
      }
      return closest.length < k;
    };

    const m = this.bucketIndex(pub);
    let l = m - 1;
    let r = m + 1;

    fill(m);
    while ((l >= 0 && fill(l)) || (r < this.buckets.length && fill(r))) {
      [l, r] = [l - 1, r + 1];
    }

    return closest;
  }
}
