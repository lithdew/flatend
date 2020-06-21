"use strict";
var __awaiter =
  (this && this.__awaiter) ||
  function (thisArg, _arguments, P, generator) {
    function adopt(value) {
      return value instanceof P
        ? value
        : new P(function (resolve) {
            resolve(value);
          });
    }
    return new (P || (P = Promise))(function (resolve, reject) {
      function fulfilled(value) {
        try {
          step(generator.next(value));
        } catch (e) {
          reject(e);
        }
      }
      function rejected(value) {
        try {
          step(generator["throw"](value));
        } catch (e) {
          reject(e);
        }
      }
      function step(result) {
        result.done
          ? resolve(result.value)
          : adopt(result.value).then(fulfilled, rejected);
      }
      step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
  };
var __asyncValues =
  (this && this.__asyncValues) ||
  function (o) {
    if (!Symbol.asyncIterator)
      throw new TypeError("Symbol.asyncIterator is not defined.");
    var m = o[Symbol.asyncIterator],
      i;
    return m
      ? m.call(o)
      : ((o =
          typeof __values === "function" ? __values(o) : o[Symbol.iterator]()),
        (i = {}),
        verb("next"),
        verb("throw"),
        verb("return"),
        (i[Symbol.asyncIterator] = function () {
          return this;
        }),
        i);
    function verb(n) {
      i[n] =
        o[n] &&
        function (v) {
          return new Promise(function (resolve, reject) {
            (v = o[n](v)), settle(resolve, reject, v.done, v.value);
          });
        };
    }
    function settle(resolve, reject, d, v) {
      Promise.resolve(v).then(function (v) {
        resolve({ value: v, done: d });
      }, reject);
    }
  };
var __importStar =
  (this && this.__importStar) ||
  function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null)
      for (var k in mod)
        if (Object.hasOwnProperty.call(mod, k)) result[k] = mod[k];
    result["default"] = mod;
    return result;
  };
var __importDefault =
  (this && this.__importDefault) ||
  function (mod) {
    return mod && mod.__esModule ? mod : { default: mod };
  };
Object.defineProperty(exports, "__esModule", { value: true });
require("mocha");
const net = __importStar(require("net"));
const events = __importStar(require("events"));
const src_1 = require("../src");
const chai_1 = __importStar(require("chai"));
const stream_1 = require("stream");
const session_1 = require("../src/session");
const tweetnacl_1 = __importDefault(require("tweetnacl"));
const kademlia_1 = require("../src/kademlia");
const stream_2 = require("../src/stream");
const net_1 = require("../src/net");
chai_1.default.use(require("chai-bytes"));
const createEndToEnd = () =>
  __awaiter(void 0, void 0, void 0, function* () {
    const server = net.createServer();
    server.listen();
    yield events.once(server, "listening");
    const info = server.address();
    const client = net.createConnection(info.port, info.address);
    const [[conn]] = yield Promise.all([
      events.once(server, "connection"),
      events.once(client, "connect"),
    ]);
    return [server, client, conn];
  });
describe("length prefix", function () {
  it("should work end-to-end", () =>
    __awaiter(this, void 0, void 0, function* () {
      const expected = [...Array(100)].map(() => Math.random().toString(16));
      const [server, client, conn] = yield createEndToEnd();
      setImmediate(() =>
        __awaiter(this, void 0, void 0, function* () {
          for (const data of expected) {
            chai_1
              .expect(client.write(stream_2.prefixLength(data)))
              .to.equal(true);
          }
          client.end();
          yield events.once(client, "close");
        })
      );
      const stream = stream_2.lengthPrefixed(conn);
      for (const data of expected) {
        chai_1
          .expect((yield stream.next()).value)
          .to.equalBytes(Buffer.from(data));
      }
      yield events.once(conn, "close");
      server.close();
      yield events.once(server, "close");
    }));
});
const endToEndHandshake = (client, conn) =>
  __awaiter(void 0, void 0, void 0, function* () {
    return yield Promise.all([
      session_1.clientHandshake(client),
      session_1.serverHandshake(conn),
    ]);
  });
describe("session", function () {
  it("should work end-to-end", () =>
    __awaiter(this, void 0, void 0, function* () {
      const expected = [...Array(100)].map(() => Math.random().toString(16));
      const [server, client, conn] = yield createEndToEnd();
      const [clientSecret, serverSecret] = yield endToEndHandshake(
        client,
        conn
      );
      chai_1.expect(clientSecret).to.equalBytes(serverSecret);
      setImmediate(() =>
        __awaiter(this, void 0, void 0, function* () {
          const session = new session_1.Session(clientSecret);
          const stream = session.decrypted(stream_2.lengthPrefixed(client));
          for (const data of expected) {
            client.write(stream_2.prefixLength(session.encrypt(data)));
          }
          for (const data of expected) {
            chai_1
              .expect((yield stream.next()).value)
              .to.equalBytes(Buffer.from(data));
          }
          client.end();
          yield events.once(client, "close");
        })
      );
      const session = new session_1.Session(serverSecret);
      const stream = session.decrypted(stream_2.lengthPrefixed(conn));
      for (const data of expected) {
        chai_1
          .expect((yield stream.next()).value)
          .to.equalBytes(Buffer.from(data));
      }
      for (const data of expected) {
        conn.write(stream_2.prefixLength(session.encrypt(data)));
      }
      yield events.once(conn, "close");
      server.close();
      yield events.once(server, "close");
    }));
});
describe("encrypted rpc", function () {
  it("should work end-to-end", () =>
    __awaiter(this, void 0, void 0, function* () {
      const expected = [...Array(100)].map(() => Math.random().toString(16));
      const [server, client, conn] = yield createEndToEnd();
      const [clientSecret, serverSecret] = yield endToEndHandshake(
        client,
        conn
      );
      chai_1.expect(clientSecret).to.equalBytes(serverSecret);
      setImmediate(() =>
        __awaiter(this, void 0, void 0, function* () {
          const rpc = new stream_2.RPC(true);
          const session = new session_1.Session(clientSecret);
          const stream = rpc.parse(
            session.decrypted(stream_2.lengthPrefixed(client))
          );
          setImmediate(() =>
            __awaiter(this, void 0, void 0, function* () {
              while (true) {
                const item = yield stream.next();
                if (item.done) {
                  break;
                }
                const { seq, frame } = item.value;
                client.write(
                  stream_2.prefixLength(
                    session.encrypt(
                      rpc.message(
                        seq,
                        Buffer.concat([Buffer.from("FROM CLIENT: "), frame])
                      )
                    )
                  )
                );
              }
            })
          );
          for (const data of expected) {
            const [req, res] = rpc.request(data);
            client.write(stream_2.prefixLength(session.encrypt(req)));
            chai_1
              .expect(yield res)
              .to.equalBytes(Buffer.from("FROM SERVER: " + data));
          }
          client.end();
          yield events.once(client, "close");
        })
      );
      const rpc = new stream_2.RPC(false);
      const session = new session_1.Session(serverSecret);
      const stream = rpc.parse(
        session.decrypted(stream_2.lengthPrefixed(conn))
      );
      setImmediate(() =>
        __awaiter(this, void 0, void 0, function* () {
          while (true) {
            const item = yield stream.next();
            if (item.done) {
              break;
            }
            const { seq, frame } = item.value;
            conn.write(
              stream_2.prefixLength(
                session.encrypt(
                  rpc.message(
                    seq,
                    Buffer.concat([Buffer.from("FROM SERVER: "), frame])
                  )
                )
              )
            );
          }
        })
      );
      for (const data of expected) {
        const [req, res] = rpc.request(data);
        conn.write(stream_2.prefixLength(session.encrypt(req)));
        chai_1
          .expect(yield res)
          .to.equalBytes(Buffer.from("FROM CLIENT: " + data));
      }
      yield events.once(conn, "close");
      server.close();
      yield events.once(server, "close");
    }));
});
describe("node", function () {
  it("should work end-to-end", () =>
    __awaiter(this, void 0, void 0, function* () {
      const aliceAddr = yield net_1.getAvailableAddress();
      const bobAddr = yield net_1.getAvailableAddress();
      const alice = new src_1.Node();
      const bob = new src_1.Node();
      alice.keys = tweetnacl_1.default.sign.keyPair();
      bob.keys = tweetnacl_1.default.sign.keyPair();
      alice.id = new kademlia_1.ID(
        alice.keys.publicKey,
        aliceAddr.host,
        aliceAddr.port
      );
      bob.id = new kademlia_1.ID(
        bob.keys.publicKey,
        bobAddr.host,
        bobAddr.port
      );
      alice.handlers["hello_world"] = (ctx) =>
        __awaiter(this, void 0, void 0, function* () {
          chai_1.expect(ctx.id.publicKey).to.equalBytes(bob.id.publicKey);
          chai_1
            .expect(yield ctx.body())
            .to.equalBytes(Buffer.from("Bob says hi!"));
          ctx.send("Alice says hi!");
        });
      bob.handlers["hello_world"] = (ctx) =>
        __awaiter(this, void 0, void 0, function* () {
          chai_1.expect(ctx.id.publicKey).to.equalBytes(alice.id.publicKey);
          chai_1
            .expect(yield ctx.body())
            .to.equalBytes(Buffer.from("Alice says hi!"));
          ctx.send("Bob says hi!");
        });
      yield bob.listen({ port: bobAddr.port });
      yield alice.connect({
        host: bobAddr.host.toString(),
        port: bobAddr.port,
      });
      const aliceToBob = () =>
        __awaiter(this, void 0, void 0, function* () {
          var e_1, _a;
          for (let i = 0; i < 10; i++) {
            const res = yield alice.push(
              ["hello_world"],
              {},
              stream_1.Readable.from("Alice says hi!")
            );
            try {
              for (
                var _b = __asyncValues(res.body), _c;
                (_c = yield _b.next()), !_c.done;

              ) {
                const chunk = _c.value;
                chai_1.expect(chunk).to.equalBytes(Buffer.from("Bob says hi!"));
              }
            } catch (e_1_1) {
              e_1 = { error: e_1_1 };
            } finally {
              try {
                if (_c && !_c.done && (_a = _b.return)) yield _a.call(_b);
              } finally {
                if (e_1) throw e_1.error;
              }
            }
          }
        });
      const bobToAlice = () =>
        __awaiter(this, void 0, void 0, function* () {
          var e_2, _d;
          for (let i = 0; i < 10; i++) {
            const res = yield bob.push(
              ["hello_world"],
              {},
              stream_1.Readable.from("Bob says hi!")
            );
            try {
              for (
                var _e = __asyncValues(res.body), _f;
                (_f = yield _e.next()), !_f.done;

              ) {
                const chunk = _f.value;
                chai_1
                  .expect(chunk)
                  .to.equalBytes(Buffer.from("Alice says hi!"));
              }
            } catch (e_2_1) {
              e_2 = { error: e_2_1 };
            } finally {
              try {
                if (_f && !_f.done && (_d = _e.return)) yield _d.call(_e);
              } finally {
                if (e_2) throw e_2.error;
              }
            }
          }
        });
      yield Promise.all([aliceToBob(), bobToAlice()]);
      yield Promise.all([alice.shutdown(), bob.shutdown()]);
    }));
});
