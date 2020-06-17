# todo

Ye 'ole todo list example using SQLite.

```
$ flatend
2020/06/17 01:27:03 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 01:27:03 Listening for HTTP requests on '[::]:3000'.
2020/06/17 01:27:10 ??? has connected to you. Services: [all_todos add_todo remove_todo done_todo]

$ go run main.go
2020/06/17 01:27:10 You are now connected to 127.0.0.1:9000. Services: []

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

```go
package main

import (
	"database/sql"
	"encoding/json"
	"github.com/lithdew/flatend"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"os/signal"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	db, err := sql.Open("sqlite3", ":memory:")
	check(err)

	_, err = db.Exec("CREATE TABLE todo (id INTEGER PRIMARY KEY AUTOINCREMENT, content TEXT, state TEXT)")
	check(err)

	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"all_todos": func(ctx *flatend.Context) {
				rows, err := db.Query("SELECT id, content, state FROM todo")
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}
				defer rows.Close()

				type Todo struct {
					ID int `json:"id"`
					Content string `json:"content"`
					State string `json:'state'`
				}

				todos := make([]Todo, 0)

				for rows.Next() {
					var todo Todo
					if err := rows.Scan(&todo.ID, &todo.Content, &todo.State); err != nil {
						ctx.Write([]byte(err.Error()))
						return
					}
					todos = append(todos, todo)
				}

				buf, err := json.Marshal(todos)
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}

				ctx.WriteHeader("Content-Type", "application/json")
				ctx.Write(buf)
			},
			"add_todo": func(ctx *flatend.Context) {
				content := ctx.Headers["params.content"]

				res, err := db.Exec("INSERT INTO todo (content, state) VALUES (?, '')", content)
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}

				var result struct {
					LastInsertID int64 `json:"lastInsertRowid"`
					RowsAffected int64 `json:"changes"`
				}

				result.LastInsertID, _ = res.LastInsertId()
				result.RowsAffected, _ = res.RowsAffected()

				buf, err := json.Marshal(result)
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}

				ctx.WriteHeader("Content-Type", "application/json")
				ctx.Write(buf)
			},
			"remove_todo": func(ctx *flatend.Context) {
				id := ctx.Headers["params.id"]

				res, err := db.Exec("DELETE FROM todo WHERE id = ?", id)
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}

				var result struct {
					LastInsertID int64 `json:"lastInsertRowid"`
					RowsAffected int64 `json:"changes"`
				}

				result.LastInsertID, _ = res.LastInsertId()
				result.RowsAffected, _ = res.RowsAffected()

				buf, err := json.Marshal(result)
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}

				ctx.WriteHeader("Content-Type", "application/json")
				ctx.Write(buf)
			},
			"done_todo": func(ctx *flatend.Context) {
				id := ctx.Headers["params.id"]

				res, err := db.Exec("UPDATE todo SET state = 'done' WHERE id = ?", id)
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}

				var result struct {
					LastInsertID int64 `json:"lastInsertRowid"`
					RowsAffected int64 `json:"changes"`
				}

				result.LastInsertID, _ = res.LastInsertId()
				result.RowsAffected, _ = res.RowsAffected()

				buf, err := json.Marshal(result)
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}

				ctx.WriteHeader("Content-Type", "application/json")
				ctx.Write(buf)
			},
		},
	}
	check(node.Start("127.0.0.1:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
	check(db.Close())
}
```