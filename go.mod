module github.com/lithdew/flatend

go 1.14

replace github.com/lithdew/kademlia => ../kademlia

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/davecgh/go-spew v1.1.1
	github.com/jpillora/backoff v1.0.0
	github.com/json-iterator/go v1.1.10
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lithdew/bytesutil v0.0.0-20200409052507-d98389230a59
	github.com/lithdew/kademlia v0.0.0-20200607181215-ff07ba2ac940
	github.com/lithdew/monte v0.0.0-20200613034704-f6d482652dc3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.0 // indirect
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	golang.org/x/tools v0.0.0-20200522201501-cb1345f3a375 // indirect
)
