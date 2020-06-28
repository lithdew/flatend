import { Duplex, finished } from "stream";
import { Stream, STREAM_CHUNK_SIZE } from "./stream";
import { ID } from "./kademlia";
import util from "util";
import { DataPacket, Opcode, ServiceResponsePacket } from "./packet";
import { Provider } from "./provider";
import { chunkBuffer } from "./node";

export type Handler = (ctx: Context) => void;

export class Context extends Duplex {
  _provider: Provider;
  _stream: Stream;
  _headersWritten = false;
  _headers: { [key: string]: string } = {};

  headers: { [key: string]: string };

  get id(): ID {
    return this._provider.id!;
  }

  constructor(
    provider: Provider,
    stream: Stream,
    headers: { [key: string]: string }
  ) {
    super();

    this._provider = provider;
    this._stream = stream;
    this.headers = headers;

    // pipe stream body to context

    setImmediate(async () => {
      for await (const frame of this._stream.body) {
        this.push(frame);
      }
      this.push(null);
    });

    // write stream eof when stream writable is closed

    setImmediate(async () => {
      await util.promisify(finished)(this, { readable: false });

      await this._writeHeader();

      const payload = new DataPacket(this._stream.id, Buffer.of()).encode();
      await this._provider.write(
        this._provider.rpc.message(
          0,
          Buffer.concat([Buffer.of(Opcode.Data), payload])
        )
      );
    });
  }

  header(key: string, val: string): Context {
    this._headers[key] = val;
    return this;
  }

  send(data: string | Buffer | Uint8Array) {
    this.write(data);
    if (!this.writableEnded) this.end();
  }

  json(data: any) {
    this.header("content-type", "application/json");
    this.send(JSON.stringify(data));
  }

  _read(size: number) {
    this._stream.body._read(size);
  }

  async body(opts?: { limit?: number }): Promise<Buffer> {
    const limit = opts?.limit ?? 2 ** 16;

    let buf = Buffer.of();
    for await (const chunk of this) {
      buf = Buffer.concat([buf, chunk]);
      if (buf.byteLength > limit) {
        throw new Error(
          `Exceeded max allowed body size limit of ${limit} byte(s).`
        );
      }
    }

    return buf;
  }

  async _writeHeader() {
    if (!this._headersWritten) {
      this._headersWritten = true;

      const payload = new ServiceResponsePacket(
        this._stream.id,
        true,
        this._headers
      ).encode();
      await this._provider.write(
        this._provider.rpc.message(
          0,
          Buffer.concat([Buffer.of(Opcode.ServiceResponse), payload])
        )
      );
    }
  }

  _write(
    chunk: any,
    encoding: BufferEncoding,
    callback: (error?: Error | null) => void
  ) {
    const write = async () => {
      await this._writeHeader();

      const buf = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk, encoding);

      for (const chunk of chunkBuffer(buf, STREAM_CHUNK_SIZE)) {
        const payload = new DataPacket(this._stream.id, chunk).encode();
        await this._provider.write(
          this._provider.rpc.message(
            0,
            Buffer.concat([Buffer.of(Opcode.Data), payload])
          )
        );
      }
    };

    write()
      .then(() => callback())
      .catch((error) => callback(error));
  }
}
