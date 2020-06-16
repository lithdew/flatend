# pipe

Whatever comes in must come out. Simple piping example: upload a file to POST /pipe.

```
$ flatend
2020/06/17 01:07:12 Listening for microservices on '127.0.0.1:9000'.
2020/06/17 01:07:12 Listening for HTTP requests on '[::]:3000'.
2020/06/17 01:07:17 ??? has connected to you. Services: [pipe]

$ go run main.go
2020/06/17 01:07:17 You are now connected to 127.0.0.1:9000. Services: []

$ POST /pipe (1.6MiB GIF)
```