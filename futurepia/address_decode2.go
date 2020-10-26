package futurepia

import (
	"github.com/blocktree/go-owcdrivers/addressEncoder"
	"github.com/blocktree/openwallet/v2/openwallet"
)

var (
	Default = AddressDecoderV2{}
)

//AddressDecoderV2
type AddressDecoderV2 struct {
	*openwallet.AddressDecoderV2Base
	wm *WalletManager //钱包管理者
}

func NewAddressDecoder2(wm *WalletManager) *AddressDecoderV2 {
	decoder := AddressDecoderV2{}
	decoder.wm = wm
	return &decoder
}

//AddressEncode 地址编码
func (dec *AddressDecoderV2) AddressEncode(hash []byte, opts ...interface{}) (string, error) {
	PIAPublicKeyPrefixCompat := dec.wm.Config.AddressPrefix
	PIA_mainnetPublic := addressEncoder.AddressType{"eos", addressEncoder.BTCAlphabet, "ripemd160", "", 33, []byte(PIAPublicKeyPrefixCompat), nil}
	address := addressEncoder.AddressEncode(hash, PIA_mainnetPublic)
	return address, nil
}

// AddressVerify 地址校验
func (dec *AddressDecoderV2) AddressVerify(address string, opts ...interface{}) bool {
	_, err := dec.wm.Api.GetBalance(address, dec.wm.Config.FeeString)
	if err == nil {
		return true
	} else {
		return false
	}
}
