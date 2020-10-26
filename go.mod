module github.com/blocktree/futurepia-adapter

go 1.14

require (
	github.com/asdine/storm v2.1.2+incompatible
	github.com/astaxie/beego v1.12.0
	github.com/blocktree/go-owcdrivers v1.2.0
	github.com/blocktree/go-owcrypt v1.1.1
	github.com/blocktree/openwallet/v2 v2.0.10
	github.com/eoscanada/eos-go v0.8.10
	github.com/imroc/req v0.2.4
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v0.0.0-20200105231215-408a2507e114
	github.com/tidwall/gjson v1.3.5
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b // indirect

)

//replace github.com/blocktree/openwallet => ../../openwallet
