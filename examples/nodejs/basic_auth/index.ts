import { Node } from "./node"
import { SessionStore } from "./session-store"
import { Memory } from "./session-store/memory"
import { randomBytes } from "tweetnacl";
import bcrypt from "bcryptjs";

const salt = bcrypt.genSaltSync(10);
const session = new SessionStore(new Memory());

const users = []

const authenticate = (id: string, password: string) => {
  return users.find(u => u.id === id)
}

const main = async () => {
  const node = await Node.start({
    addrs: [`127.0.0.1:9000`],
    services: {
      login: (ctx) => {
        if (ctx.headers["params.id"] && ctx.headers["params.password"]) {
          const authenticated = authenticate(ctx.headers["params.id"], ctx.headers["params.password"]);
          if (authenticated && bcrypt.compareSync(ctx.headers["params.password"], authenticated.password)) {
            const sid = Buffer.from(randomBytes(32)).toString('hex');
            session.create(sid, authenticated);
            ctx.json(session.store.get(sid));
          }
          else {
            ctx.json({
              err: "Invalid credentials"
            });
          }
        }
        else {
          ctx.json({
            err: "Invalid credentials"
          });
        }
      },
      signup: (ctx) => {
        if (ctx.headers["params.id"] && ctx.headers["params.password"]) {
          users.push({
            id: ctx.headers["params.id"],
            password: bcrypt.hashSync(ctx.headers["params.password"], salt)
          })
          ctx.send('done');
        }
      },
      logout: (ctx) => {
        const sess = session.load(ctx.headers)
        session.clear(sess.id)
      }
    },
  });
};

main().catch((err) => console.error(err));
