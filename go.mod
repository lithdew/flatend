module github.com/lithdew/flatend

go 1.14

replace github.com/lithdew/kademlia => ../kademlia

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/davecgh/go-spew v1.1.1
	github.com/jpillora/backoff v1.0.0
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lithdew/bytesutil v0.0.0-20200409052507-d98389230a59
	github.com/lithdew/kademlia v0.0.0-20200607181215-ff07ba2ac940
	github.com/lithdew/monte v0.0.0-20200611093340-15ff088304c9
	github.com/oasislabs/ed25519 v0.0.0-20200302143042-29f6767a7c3e
	github.com/stretchr/testify v1.6.0
	github.com/valyala/bytebufferpool v1.0.0
	go.uber.org/goleak v1.0.0
	golang.org/x/crypto v0.0.0-20191119213627-4f8c1d86b1ba
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	golang.org/x/tools v0.0.0-20200522201501-cb1345f3a375 // indirect
)
