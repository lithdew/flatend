import { Buffer } from "https://deno.land/std/node/buffer.ts";
import { ID } from "./kademlia.ts";
import * as net from "./std-node-net.ts";
import { Session } from "./session.ts";
import {
  drain,
  lengthPrefixed,
  prefixLength,
  RPC,
  Stream,
  Streams,
} from "./stream.ts";
import { Opcode } from "./packet.ts";

export class Provider {
  id?: ID;
  handshaked = false;
  services = new Set<string>();

  sock: net.Socket;
  session: Session;
  rpc: RPC;
  streams: Streams;

  get addr(): string {
    if (this.id) return this.id.addr;
    return "<anon>";
  }

  constructor(sock: net.Socket, session: Session, client: boolean) {
    this.sock = sock;
    this.session = session;
    this.rpc = new RPC(client);
    this.streams = new Streams(client);
  }

  async write(buf: Buffer) {
    buf = prefixLength(this.session.encrypt(buf));
    if (!this.sock.write(buf)) await drain(this.sock);
  }

  async request(data: Buffer): Promise<Buffer> {
    const [req, res] = this.rpc.request(data);
    await this.write(req);
    return await res;
  }

  async *read() {
    const stream = this.rpc.parse(
      this.session.decrypted(lengthPrefixed(this.sock))
    );
    for await (let { seq, frame } of stream) {
      if (frame.byteLength < 1)
        throw new Error(`Frame must be prefixed with an opcode byte.`);

      const opcode = frame.readUInt8();
      frame = frame.slice(1);

      yield { seq, opcode, frame };
    }
  }

  async push(
    services: string[],
    headers: { [key: string]: string },
    body: AsyncIterable<Buffer>
  ): Promise<Stream> {
    const err = new Error(
      `No nodes were able to process your request for service(s): [${services.join(
        ", "
      )}]`
    );

    const stream = this.streams.register();
    const [header, handled] = this.streams.push(stream, services, headers);

    await this.write(
      this.rpc.message(
        0,
        Buffer.concat([Buffer.of(Opcode.ServiceRequest), header])
      )
    );

    for await (const chunk of this.streams.encoded(stream.id, body)) {
      await this.write(
        this.rpc.message(0, Buffer.concat([Buffer.of(Opcode.Data), chunk]))
      );
    }

    if (!(await handled)) {
      throw err;
    }

    return stream;
  }
}
