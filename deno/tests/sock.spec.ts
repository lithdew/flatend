import "mocha";
import * as net from "net";
import * as events from "events";
import { Node } from "../src";
import chai, { expect } from "chai";
import { Readable } from "stream";
import { clientHandshake, serverHandshake, Session } from "../src/session";
import * as nacl from "tweetnacl";
import { ID } from "../src/kademlia";
import { lengthPrefixed, prefixLength, RPC } from "../src/stream";
import { Context } from "../src/context";
import { getAvailableAddress } from "../src/net";

chai.use(require("chai-bytes"));

const createEndToEnd = async (): Promise<
  [net.Server, net.Socket, net.Socket]
> => {
  const server = net.createServer();
  server.listen();

  await events.once(server, "listening");

  const info = (<net.AddressInfo>server.address())!;
  const client = net.createConnection(info.port, info.address);

  const [[conn]] = <[[net.Socket], any[]]>(
    await Promise.all([
      events.once(server, "connection"),
      events.once(client, "connect"),
    ])
  );

  return [server, client, conn];
};

describe("length prefix", function () {
  it("should work end-to-end", async () => {
    const expected = [...Array(100)].map(() => Math.random().toString(16));

    const [server, client, conn] = await createEndToEnd();

    setImmediate(async () => {
      for (const data of expected) {
        expect(client.write(prefixLength(data))).to.equal(true);
      }
      client.end();
      await events.once(client, "close");
    });

    const stream = lengthPrefixed(conn);
    for (const data of expected) {
      expect((await stream.next()).value).to.equalBytes(Buffer.from(data));
    }
    await events.once(conn, "close");

    server.close();
    await events.once(server, "close");
  });
});

const endToEndHandshake = async (
  client: net.Socket,
  conn: net.Socket
): Promise<[Uint8Array, Uint8Array]> => {
  return await Promise.all([clientHandshake(client), serverHandshake(conn)]);
};

describe("session", function () {
  it("should work end-to-end", async () => {
    const expected = [...Array(100)].map(() => Math.random().toString(16));

    const [server, client, conn] = await createEndToEnd();
    const [clientSecret, serverSecret] = await endToEndHandshake(client, conn);

    expect(clientSecret).to.equalBytes(serverSecret);

    setImmediate(async () => {
      const session = new Session(clientSecret);
      const stream = session.decrypted(lengthPrefixed(client));

      for (const data of expected) {
        client.write(prefixLength(session.encrypt(data)));
      }

      for (const data of expected) {
        expect((await stream.next()).value).to.equalBytes(Buffer.from(data));
      }

      client.end();
      await events.once(client, "close");
    });

    const session = new Session(serverSecret);
    const stream = session.decrypted(lengthPrefixed(conn));

    for (const data of expected) {
      expect((await stream.next()).value).to.equalBytes(Buffer.from(data));
    }

    for (const data of expected) {
      conn.write(prefixLength(session.encrypt(data)));
    }

    await events.once(conn, "close");

    server.close();
    await events.once(server, "close");
  });
});

describe("encrypted rpc", function () {
  it("should work end-to-end", async () => {
    const expected = [...Array(100)].map(() => Math.random().toString(16));

    const [server, client, conn] = await createEndToEnd();
    const [clientSecret, serverSecret] = await endToEndHandshake(client, conn);

    expect(clientSecret).to.equalBytes(serverSecret);

    setImmediate(async () => {
      const rpc = new RPC(true);
      const session = new Session(clientSecret);

      const stream = rpc.parse(session.decrypted(lengthPrefixed(client)));

      setImmediate(async () => {
        while (true) {
          const item = await stream.next();
          if (item.done) {
            break;
          }

          const { seq, frame } = item.value;

          client.write(
            prefixLength(
              session.encrypt(
                rpc.message(
                  seq,
                  Buffer.concat([Buffer.from("FROM CLIENT: "), frame])
                )
              )
            )
          );
        }
      });

      for (const data of expected) {
        const [req, res] = rpc.request(data);
        client.write(prefixLength(session.encrypt(req)));
        expect(await res).to.equalBytes(Buffer.from("FROM SERVER: " + data));
      }

      client.end();
      await events.once(client, "close");
    });

    const rpc = new RPC(false);
    const session = new Session(serverSecret);

    const stream = rpc.parse(session.decrypted(lengthPrefixed(conn)));

    setImmediate(async () => {
      while (true) {
        const item = await stream.next();
        if (item.done) {
          break;
        }

        const { seq, frame } = item.value;

        conn.write(
          prefixLength(
            session.encrypt(
              rpc.message(
                seq,
                Buffer.concat([Buffer.from("FROM SERVER: "), frame])
              )
            )
          )
        );
      }
    });

    for (const data of expected) {
      const [req, res] = rpc.request(data);
      conn.write(prefixLength(session.encrypt(req)));
      expect(await res).to.equalBytes(Buffer.from("FROM CLIENT: " + data));
    }

    await events.once(conn, "close");

    server.close();
    await events.once(server, "close");
  });
});

describe("node", function () {
  it("should work end-to-end", async () => {
    const aliceAddr = await getAvailableAddress();
    const bobAddr = await getAvailableAddress();

    const alice = new Node();
    const bob = new Node();

    alice.keys = nacl.sign.keyPair();
    bob.keys = nacl.sign.keyPair();

    alice.id = new ID(alice.keys.publicKey, aliceAddr.host, aliceAddr.port);
    bob.id = new ID(bob.keys.publicKey, bobAddr.host, bobAddr.port);

    alice.handlers["hello_world"] = async (ctx: Context) => {
      expect(ctx.id.publicKey).to.equalBytes(bob.id!.publicKey!);
      expect(await ctx.body()).to.equalBytes(Buffer.from("Bob says hi!"));
      ctx.send("Alice says hi!");
    };

    bob.handlers["hello_world"] = async (ctx: Context) => {
      expect(ctx.id.publicKey).to.equalBytes(alice.id!.publicKey!);
      expect(await ctx.body()).to.equalBytes(Buffer.from("Alice says hi!"));
      ctx.send("Bob says hi!");
    };

    await bob.listen({ port: bobAddr.port });
    await alice.connect({ host: bobAddr.host.toString(), port: bobAddr.port });

    const aliceToBob = async () => {
      for (let i = 0; i < 10; i++) {
        const res = await alice.push(
          ["hello_world"],
          {},
          Readable.from("Alice says hi!")
        );
        for await (const chunk of res.body) {
          expect(chunk).to.equalBytes(Buffer.from("Bob says hi!"));
        }
      }
    };

    const bobToAlice = async () => {
      for (let i = 0; i < 10; i++) {
        const res = await bob.push(
          ["hello_world"],
          {},
          Readable.from("Bob says hi!")
        );
        for await (const chunk of res.body) {
          expect(chunk).to.equalBytes(Buffer.from("Alice says hi!"));
        }
      }
    };

    await Promise.all([aliceToBob(), bobToAlice()]);
    await Promise.all([alice.shutdown(), bob.shutdown()]);
  });
});
