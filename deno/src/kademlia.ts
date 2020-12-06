import { Buffer } from "https://deno.land/std/node/buffer.ts";
import * as nacl from "https://deno.land/x/tweetnacl_deno/src/nacl.ts";
import ipaddr from "https://jspm.dev/ipaddr.js";
import { assert } from "https://deno.land/std/testing/asserts.ts";

function BufferCompare(a: Buffer | Uint8Array, b: Buffer|Uint8Array) {
  //if (typeof a.compare === 'function') return a.compare(b)
  if (a === b) return 0

  var x = a.length
  var y = b.length

  var i = 0
  var len = Math.min(x, y)
  while (i < len) {
    if (a[i] !== b[i]) break

    ++i
  }

  if (i !== len) {
    x = a[i]
    y = b[i]
  }

  if (x < y) return -1
  if (y < x) return 1
  return 0
}

type IPv4 = ipaddr.IPv4;
type IPv6 = ipaddr.IPv6;

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
  publicKey: Uint8Array = Buffer.alloc(nacl.SignLength.PublicKey);
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
    const publicKey = Uint8Array.from(buf.slice(0, nacl.SignLength.PublicKey));
    buf = buf.slice(nacl.SignLength.PublicKey);

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
    ...Array(nacl.SignLength.PublicKey * 8),
  ].map(() => []);

  pub: Uint8Array;
  cap: number = 16;
  length: number = 0;

  public constructor(
    pub: Uint8Array = Buffer.alloc(nacl.SignLength.PublicKey)
  ) {
    this.pub = pub;
  }

  private bucketIndex(pub: Uint8Array): number {
    if (BufferCompare(pub, this.pub) === 0) return 0;
    return leadingZeros(xor(pub, this.pub));
  }

  public update(id: ID): UpdateResult {
    if (BufferCompare(id.publicKey, this.pub) === 0) return UpdateResult.Fail;

    const bucket = this.buckets[this.bucketIndex(id.publicKey)];

    const i = bucket.findIndex(
      (item) => BufferCompare(item.publicKey, id.publicKey) === 0
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
    const i = bucket.findIndex((id) => BufferCompare(id.publicKey, pub) === 0);
    if (i >= 0) {
      bucket.splice(i, 1);
      this.length--;
      return true;
    }
    return false;
  }

  public has(pub: Uint8Array): boolean {
    const bucket = this.buckets[this.bucketIndex(pub)];
    return !!bucket.find((id) => BufferCompare(id.publicKey, pub) === 0);
  }

  public closestTo(pub: Uint8Array, k = this.cap): ID[] {
    const closest: ID[] = [];

    const fill = (i: number) => {
      const bucket = this.buckets[i];
      for (let i = 0; closest.length < k && i < bucket.length; i++) {
        if (BufferCompare(bucket[i].publicKey, pub) != 0)
          closest.push(bucket[i]);
      }
    };

    const m = this.bucketIndex(pub);

    fill(m);

    for (
      let i = 1;
      closest.length < k && (m - i >= 0 || m + i < this.buckets.length);
      i++
    ) {
      if (m - i >= 0) fill(m - i);
      if (m + i < this.buckets.length) fill(m + i);
    }

    closest.sort((a: ID, b: ID) =>
      BufferCompare(xor(a.publicKey, pub), xor(b.publicKey, pub))
    );

    return closest.length > k ? closest.slice(0, k) : closest;
  }
}
