import { Readable, Writable } from "stream";
import events, { EventEmitter } from "events";
import { DataPacket, ServiceRequestPacket } from "./packet";

export async function drain(writable: Writable) {
  if (writable.destroyed) throw new Error(`premature close`);

  await Promise.race([
    events.once(writable, "drain"),
    events.once(writable, "close").then(() => {
      throw new Error(`premature close`);
    }),
  ]);
}

export async function* lengthPrefixed(stream: AsyncIterable<Buffer>) {
  let buf: Buffer = Buffer.of();
  let size: number | undefined;

  for await (const data of stream) {
    buf = Buffer.concat([buf, data]);

    while (true) {
      if (!size) {
        if (buf.byteLength < 4) break;
        size = buf.readUInt32BE(0);
        buf = buf.slice(4);
      }

      if (buf.byteLength < size) break;

      const frame = buf.slice(0, size);
      buf = buf.slice(size);
      size = undefined;

      yield frame;
    }
  }
}

export function prefixLength(src: string | ArrayBufferLike): Buffer {
  const data = Buffer.isBuffer(src) ? src : Buffer.from(src);

  const header = Buffer.alloc(4);
  header.writeUInt32BE(data.byteLength);
  return Buffer.concat([header, data]);
}

export class RPC {
  pending: EventEmitter = new EventEmitter();
  initial: number;
  counter: number;

  constructor(client: boolean) {
    this.counter = this.initial = client ? 1 : 2;
  }

  next(): number {
    const seq = this.counter;
    if ((this.counter += 2) === 0) {
      this.counter = this.initial;
    }
    return seq;
  }

  message(seq: number, src: string | ArrayBufferLike): Buffer {
    const buf = Buffer.isBuffer(src) ? src : Buffer.from(src);
    const header = Buffer.alloc(4);
    header.writeUInt32BE(seq);
    return Buffer.concat([header, buf]);
  }

  request(src: string | ArrayBufferLike): [Buffer, Promise<Buffer>] {
    const seq = this.next();
    const buf = this.message(seq, src);
    return [buf, this.wait(seq)];
  }

  async wait(seq: number): Promise<Buffer> {
    return (<[Buffer]>await events.once(this.pending, `${seq}`))[0];
  }

  async *parse(stream: AsyncIterable<Buffer>) {
    for await (let frame of stream) {
      if (frame.byteLength < 4)
        throw new Error(
          `Frame must be prefixed with an unsigned 32-bit sequence number.`
        );

      const seq = frame.readUInt32BE();
      frame = frame.slice(4);

      if (seq !== 0 && seq % 2 === this.initial % 2) {
        this.pending.emit(`${seq}`, frame);
        continue;
      }

      yield { seq, frame };
    }
  }
}

export const STREAM_CHUNK_SIZE = 2048;

export class Stream extends EventEmitter {
  id: number;
  body: Readable;
  headers?: { [key: string]: string };

  constructor(id: number) {
    super();

    this.id = id;

    this.body = new Readable();
    this.body._read = () => {
      return;
    };
  }
}

export class Streams {
  active = new Map<number, Stream>();
  initial: number;
  counter: number;

  constructor(client: boolean) {
    this.counter = this.initial = client ? 0 : 1;
  }

  register(id?: number): Stream {
    if (id === undefined) {
      id = this.counter;
      if ((this.counter += 2) === 0) {
        this.counter = this.initial;
      }
    }

    if (this.active.has(id)) {
      throw new Error(
        `Attempted to register stream with ID ${id} which already exists.`
      );
    }

    const stream = new Stream(id);
    this.active.set(id, stream);

    return stream;
  }

  get(id: number): Stream | undefined {
    return this.active.get(id);
  }

  push(
    stream: Stream,
    services: string[],
    headers: { [key: string]: string }
  ): [Buffer, Promise<Boolean>] {
    return [
      new ServiceRequestPacket(stream.id, services, headers).encode(),
      this.wait(stream),
    ];
  }

  pull(stream: Stream, handled: boolean, headers: { [key: string]: string }) {
    stream.headers = headers;
    stream.emit("ready", handled);
  }

  recv(stream: Stream, data: Buffer) {
    if (data.byteLength === 0) {
      this.active.delete(stream.id);
      stream.body.push(null);
    } else {
      stream.body.push(data);
    }
  }

  async *encoded(id: number, body: AsyncIterable<Buffer>) {
    let buf: Buffer = Buffer.of();
    for await (const chunk of body) {
      buf = Buffer.concat([buf, Buffer.from(chunk)]);
      while (buf.byteLength >= STREAM_CHUNK_SIZE) {
        yield new DataPacket(id, buf.slice(0, STREAM_CHUNK_SIZE)).encode();
        buf = buf.slice(STREAM_CHUNK_SIZE);
      }
    }
    if (buf.byteLength > 0) {
      yield new DataPacket(id, buf).encode();
    }
    yield new DataPacket(id, Buffer.of()).encode();
  }

  async wait(stream: EventEmitter): Promise<Boolean> {
    return (<[Boolean]>await events.once(stream, "ready"))[0];
  }
}
