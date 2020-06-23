import { Node } from "./node"
import { AppSession } from "./app-session"
import { randomBytes } from "tweetnacl";

const session = new AppSession();
const users = [
  {
    id: "john",
    password: "123"
  },
  {
    id: "mary",
    password: "456"
  }
]

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
          if (authenticated) {
            const sid = Buffer.from(randomBytes(32)).toString('hex');
            session.create(sid, authenticated);
            ctx.json(session.load(sid));
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
            password: ctx.headers["params.password"]
          })
          ctx.send('done');
        }
      },
      logout: (ctx) => {
        const sess = session.load(ctx.headers["params.sid"])
        session.clear(sess.id)
      }
    },
  });
};

main().catch((err) => console.error(err));
