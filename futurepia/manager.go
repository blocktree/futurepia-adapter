/*
 * Copyright 2018 The OpenWallet Authors
 * This file is part of the OpenWallet library.
 *
 * The OpenWallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The OpenWallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package futurepia

import (
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase
	Api             *Client                         // 节点客户端
	Config          *WalletConfig                   // 节点配置
	Decoder         openwallet.AddressDecoder       //地址编码器
	DecoderV2       openwallet.AddressDecoderV2     //地址编码器
	TxDecoder       openwallet.TransactionDecoder   //交易单编码器
	Log             *log.OWLogger                   //日志工具
	ContractDecoder openwallet.SmartContractDecoder //智能合约解析器
	Blockscanner    *PIABlockScanner                //区块扫描器
	CacheManager    openwallet.ICacheManager        //缓存管理器
}

func NewWalletManager() *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig(Symbol)
	wm.Api = new(Client)
	wm.Blockscanner = NewPIABlockScanner(&wm)
	wm.Decoder = NewAddressDecoder(&wm)
	wm.TxDecoder = NewTransactionDecoder(&wm)
	wm.Log = log.NewOWLogger(wm.Symbol())
	wm.DecoderV2 = NewAddressDecoder2(&wm)
	wm.ContractDecoder = NewContractDecoder(&wm)
	return &wm
}



//GetAddressDecode 地址解析器
//如果实现了AddressDecoderV2，就无需实现AddressDecoder
func (a *WalletManager) GetAddressDecoderV2() openwallet.AddressDecoderV2 {
	return a.DecoderV2
}