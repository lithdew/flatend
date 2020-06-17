import {
  DataPacket,
  FindNodeRequest,
  FindNodeResponse,
  HandshakePacket,
  Opcode,
  ServiceRequestPacket,
  ServiceResponsePacket,
} from "./packet";

import nacl from "tweetnacl";
import assert from "assert";
import { MonteSocket } from "./monte";
import * as net from "net";
import { Context } from "./context";
import { ID, Table } from "./kademlia";
import { resolve } from "./net";
import ipaddr, { IPv4, IPv6 } from "ipaddr.js";

const debug = require("debug")("flatend");

type Handler = (ctx: Context) => void;

class Provider {
  id?: ID;
  sock: MonteSocket;
  services: Set<string>;
  init: boolean = false;

  count: number = 0;
  streams: Map<number, Context> = new Map<number, Context>();

  constructor(
    id: ID | undefined,
    sock: MonteSocket,
    services = new Set<string>()
  ) {
    this.id = id;
    this.sock = sock;
    this.services = services;
  }

  get addr(): string {
    return this.id?.addr ?? "<anon>";
  }
}

interface NodeOpts {
  addrs?: string[];
  addr?: string;
  keys?: nacl.SignKeyPair;
  services?: { [key: string]: Handler };

  // bindAddrs?: string[], (not supported yet)
}

export class Node {
  #providers = new WeakMap<MonteSocket, Provider>();
  #services = new Map<string, WeakSet<Provider>>();
  #clients = new Map<string, MonteSocket>();
  #table = new Table();

  #handlers: { [key: string]: Handler } = {};

  #id: ID | undefined;
  #keys: nacl.SignKeyPair | undefined;

  get anonymous(): boolean {
    return this.#id === undefined || this.#keys === undefined;
  }

  get services(): string[] {
    return [...Object.keys(this.#handlers)];
  }

  public register(service: string, handler: Handler) {
    this.#handlers[service] = handler;
  }

  private registerProvider(
    sock: MonteSocket,
    packet: HandshakePacket
  ): [Provider, boolean] {
    let id: ID | undefined;

    if (packet.id && packet.signature) {
      if (
        !nacl.sign.detached.verify(
          packet.payload,
          packet.signature,
          packet.id.publicKey
        )
      ) {
        throw new Error(`Handshake packet signature is malformed.`);
      }
      id = packet.id;
    }

    if (id) this.#table.update(id);

    let provider = this.#providers.get(sock);
    let exists = provider && provider.id && provider.init;

    if (!exists) {
      provider!.init = true;

      sock.once("end", () => {
        if (id) this.#table.delete(id.publicKey);
        provider!.services.forEach((service) =>
          this.#services.get(service)?.delete(provider!)
        );
        this.#providers.delete(sock);
      });
    }

    provider!.id = id;
    provider!.services = new Set<string>(...packet.services);

    provider!.services.forEach((service) => {
      if (!this.#services.has(service))
        this.#services.set(service, new WeakSet<Provider>());
      this.#services.get(service)!.add(provider!);
    });

    return [provider!, !!exists];
  }

  static async start(opts?: NodeOpts): Promise<Node> {
    let bindHost: IPv4 | IPv6;
    let bindPort: number;

    let id: ID | undefined;
    let table: Table | undefined;

    if (opts?.keys) {
      if (opts?.addr) {
        [bindHost, bindPort] = await resolve(opts.addr);
      } else {
        [bindHost, bindPort] = await new Promise((resolve, reject) => {
          const server = new net.Server();
          server.unref();
          server.on("error", reject);
          server.listen(() => {
            const { address, port } = <net.AddressInfo>server.address();

            let host = ipaddr.parse(address);
            if (host.kind() === "ipv6" && (<IPv6>host).isIPv4MappedAddress()) {
              host = (<IPv6>host).toIPv4Address();
            }

            server.close(() => resolve([host, port]));
          });
        });
      }

      id = new ID(opts.keys.publicKey, bindHost, bindPort);
      table = new Table(id.publicKey);
    }

    const node = new Node();

    if (id) node.#id = id;
    if (table) node.#table = table;

    if (opts?.keys) node.#keys = opts.keys;
    if (opts?.services) node.#handlers = opts.services;

    if (opts?.addrs) {
      await Promise.all(
        opts.addrs.map(
          async (addr) => await node.probe(await node.getClient(addr))
        )
      );
      await node.bootstrap();
    }

    return node;
  }

  private async bootstrap() {
    const pub = this.#id?.publicKey ?? Buffer.alloc(nacl.sign.publicKeyLength);
    const visited = new Set<string>();
    const closest: ID[] = [];

    let queue: ID[] = this.#table.closestTo(pub, this.#table.cap);
    if (queue.length === 0) {
      return;
    }

    queue.forEach((id) =>
      visited.add(Buffer.from(id.publicKey).toString("hex"))
    );

    while (queue.length > 0) {
      const next: ID[] = [];

      await Promise.all(
        queue.map(async (id) => {
          const client = await this.getClient(id.addr);

          await this.probe(client);

          const res = FindNodeResponse.decode(
            await client.request(
              Buffer.concat([
                Buffer.of(Opcode.FindNodeRequest),
                new FindNodeRequest(pub).encode(),
              ])
            )
          )[0];

          res.closest = res.closest.filter((id) => {
            return !visited.has(Buffer.from(id.publicKey).toString("hex"));
          });

          closest.push(...res.closest);
          next.push(...res.closest);
        })
      );

      queue = next;
    }

    debug(`Discovered ${closest.length} peer(s).`);
  }

  private createHandshakePacket(): HandshakePacket {
    let packet = new HandshakePacket(null, this.services, null);
    if (!this.anonymous) {
      packet.id = this.#id!;
      packet.signature = Buffer.from(
        nacl.sign(packet.payload, this.#keys!.secretKey)
      );
    }
    return packet;
  }

  private async getClient(addr: string) {
    const [host, port] = await resolve(addr);

    let sock = this.#clients.get(addr);
    if (!sock) {
      sock = await MonteSocket.connect({ host: host.toString(), port });

      this.#providers.set(sock, new Provider(undefined, sock));
      this.#clients.set(addr, sock);

      sock.once("end", () => this.#clients.delete(addr));
      sock.on("data", this._data.bind(this));
      sock.on("error", debug);
    }

    return sock;
  }

  private async probe(sock: MonteSocket) {
    const packet = HandshakePacket.decode(
      await sock.request(
        Buffer.concat([
          Buffer.of(Opcode.Handshake),
          this.createHandshakePacket().encode(),
        ])
      )
    )[0];

    const [provider, exists] = this.registerProvider(sock, packet);

    if (!exists) {
      debug(
        `You are now connected to ${
          provider.addr
        }. Services: [${packet.services.join(", ")}]`
      );

      sock.once("end", () => {
        let count = 8;

        const reconnect = async () => {
          if (count-- === 0) {
            debug(`Tried 8 times reconnecting to ${provider.addr}. Giving up.`);
            return;
          }
          debug(`Trying to reconnect to ${provider.addr}. Sleeping for 500ms.`);
          try {
            await this.probe(await this.getClient(provider.addr));
          } catch (err) {
            setTimeout(reconnect, 500);
          }
        };
        setTimeout(reconnect, 500);
      });
    }
  }

  private _data({
    sock,
    seq,
    body,
  }: {
    sock: MonteSocket;
    seq: number;
    body: Buffer;
  }) {
    const opcode: Opcode = body.readUInt8();
    body = body.slice(1);

    switch (opcode) {
      case Opcode.Handshake: {
        const packet = HandshakePacket.decode(body)[0];

        this.#providers.set(sock, new Provider(undefined, sock));
        const [provider, exists] = this.registerProvider(sock, packet);

        if (!exists) {
          debug(
            `${provider.addr} has connected to you. Services: [${[
              ...provider.services,
            ].join(", ")}]`
          );
        }

        sock.send(seq, this.createHandshakePacket().encode());

        return;
      }
      case Opcode.ServiceRequest: {
        const packet = ServiceRequestPacket.decode(body)[0];

        const provider = this.#providers.get(sock);
        assert(provider, "Socket is not registered as a provider.");

        assert(
          !provider.streams.has(packet.id),
          `Stream with ID ${packet.id} already exists.`
        );

        const service = packet.services.find(
          (service) => service in this.#handlers
        );
        if (!service) {
          const response = new ServiceResponsePacket(packet.id, false, {});
          sock.send(
            0,
            Buffer.concat([
              Buffer.of(Opcode.ServiceResponse),
              response.encode(),
            ])
          );
          return;
        }

        const ctx = new Context(packet.id, sock, packet.headers);
        provider.streams.set(packet.id, ctx);

        const handler = this.#handlers[service];

        (async () => {
          try {
            await handler(ctx);
          } catch (err) {
            ctx.json({ error: err.message });
          }
        })();

        return;
      }
      case Opcode.Data: {
        const packet = DataPacket.decode(body)[0];

        const provider = this.#providers.get(sock);
        assert(provider, "Socket is not registered as a provider.");

        const ctx = provider.streams.get(packet.id);
        assert(
          ctx,
          `Got data packet with stream ID ${packet.id}, but stream does not exist.`
        );

        if (packet.data.byteLength > 0) {
          ctx.push(packet.data);
          return;
        }

        provider.streams.delete(packet.id);
        ctx.push(null);

        return;
      }
      case Opcode.FindNodeRequest: {
        const packet = FindNodeRequest.decode(body)[0];
        sock.send(
          seq,
          new FindNodeResponse(this.#table.closestTo(packet.target)).encode()
        );
        return;
      }
      default: {
        throw new Error(`Unknown opcode ${opcode}.`);
      }
    }
  }
}
