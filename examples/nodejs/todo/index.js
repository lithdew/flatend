const {Node} = require("flatend");
const Database = require("better-sqlite3");

const db = new Database(":memory:");
db.exec(`CREATE TABLE todo (id INTEGER PRIMARY KEY AUTOINCREMENT, content TEXT, state TEXT)`);

const all = async ctx => ctx.json(await db.prepare(`SELECT id, content, state FROM todo`).all());

const add = async ctx => {
    const content = ctx.headers["params.content"];
    ctx.json(await db.prepare(`INSERT INTO todo (content, state) VALUES (?, '')`).run(content));
}

const remove = async ctx => {
    const id = ctx.headers["params.id"]
    ctx.json(await db.prepare(`DELETE FROM todo WHERE id = ?`).run(id))
}

const done = async ctx => {
    const id = ctx.headers["params.id"]
    ctx.json(await db.prepare(`UPDATE todo SET state = 'done' WHERE id = ?`).run(id));
}

const main = async () => {
    const node = new Node();

    node.register('all_todos', all);
    node.register('add_todo', add);
    node.register('remove_todo', remove);
    node.register('done_todo', done);

    await node.dial("127.0.0.1:9000");
}

main().catch(err => console.error(err));