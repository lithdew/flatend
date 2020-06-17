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
