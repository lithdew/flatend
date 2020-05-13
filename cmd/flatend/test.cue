import "net"

// SQL schema.

SQL_Database :: string
SQL_Databases :: [string]: string
SQL :: SQL_Database | SQL_Databases

sql: SQL
sql_id :: (sql & string) | or([k for k, _ in sql])

// SQL handler.

SQL_Query :: {
	db?:   sql_id
    query: string
}

// HTTP schema.

HTTP_Route :: =~"^(GET|POST) ((?:/[:a-zA-Z]+)+)$"

HTTP :: {
	host: *"" | net.IP
	port: *0 | uint16
	bind_addr: net.JoinHostPort(host, port)
	routes: {
		[HTTP_Route]: SQL_Query
	}
}

http: HTTP

// Example configuration file.

sql: {
	my_database:  "sqlite://:memory:"
	my_database2: "sqlite://:memory:"
}

http: routes: {
	"GET /post/:id": {
		db:    "my_database"
		query: "SELECT * FROM posts WHERE id = :id"
	}
	"POST /post/:id": {
		db:    "my_database2"
		query: "INSERT INTO posts(id, content) VALUES (:id, :content)"
	}
}
