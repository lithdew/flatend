# todo

Ye 'ole todo list example using SQLite.

```
$ flatend
2020/06/17 01:27:03 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 01:27:03 Listening for HTTP requests on '[::]:3000'.
2020/06/17 01:27:10 <anon> has connected to you. Services: [all_todos add_todo remove_todo done_todo]

$ DEBUG=* node index.js 
  flatend You are now connected to 127.0.0.1:9000. Services: [] +0ms
  flatend Discovered 0 peer(s). +11ms   

$ http://localhost:3000/
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /"
static = "./public"

[[http.routes]]
path = "GET /todos"
service = "all_todos"

[[http.routes]]
path = "GET /todos/add/:content"
service = "add_todo"

[[http.routes]]
path = "GET /todos/remove/:id"
service = "remove_todo"

[[http.routes]]
path = "GET /todos/done/:id"
service = "done_todo"
```

```js
const { Node } = require("flatend");
const Database = require("better-sqlite3");

const db = new Database(":memory:");
db.exec(
  `CREATE TABLE todo (id INTEGER PRIMARY KEY AUTOINCREMENT, content TEXT, state TEXT)`
);

const all = async (ctx) =>
  ctx.json(await db.prepare(`SELECT id, content, state FROM todo`).all());

const add = async (ctx) => {
  const content = ctx.headers["params.content"];
  ctx.json(
    await db
      .prepare(`INSERT INTO todo (content, state) VALUES (?, '')`)
      .run(content)
  );
};

const remove = async (ctx) => {
  const id = ctx.headers["params.id"];
  ctx.json(await db.prepare(`DELETE FROM todo WHERE id = ?`).run(id));
};

const done = async (ctx) => {
  const id = ctx.headers["params.id"];
  ctx.json(
    await db.prepare(`UPDATE todo SET state = 'done' WHERE id = ?`).run(id)
  );
};

const main = async () => {
  await Node.start({
    addrs: ["127.0.0.1:9000"],
    services: {
      all_todos: all,
      add_todo: add,
      remove_todo: remove,
      done_todo: done,
    },
  });
};

main().catch((err) => console.error(err));
```
