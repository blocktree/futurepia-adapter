package openwtester

import (
	"github.com/blocktree/futurepia-adapter/futurepia"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openw"
)

func init() {
	//注册钱包管理工具
	log.Notice("Wallet Manager Load Successfully.")
	// openw.RegAssets(eosio.Symbol, eosio.NewWalletManager(nil))

	openw.RegAssets(futurepia.Symbol, futurepia.NewWalletManager())
}
