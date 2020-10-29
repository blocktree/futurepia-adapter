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
	"fmt"
	"github.com/eoscanada/eos-go/ecc"

	"github.com/blocktree/openwallet/v2/common"
	"github.com/blocktree/openwallet/v2/crypto"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/eoscanada/eos-go"
)

// Block model
type Block struct {
	/*
		{
		    "timestamp": "2019-01-24T19:32:05.500",
		    "producer": "blkproducer1",
		    "confirmed": 0,
		    "previous": "0137c066283ef586d4e1dba4711b2ddf0248628595855361d9b0920e7f64ea92",
		    "transaction_mroot": "0000000000000000000000000000000000000000000000000000000000000000",
		    "action_mroot": "60c9f06aef01b1b4b2088785c9239c960bca8fc23cedd6b8104c69c0335a6d39",
		    "schedule_version": 2,
		    "new_producers": null,
		    "header_extensions": [],
		    "producer_signature": "SIG_K1_K11ScNfXdat71utYJtkd8E6dFtvA7qQ3ww9K74xEpFvVCyeZhXTarwvGa7QqQTRw3CLFbsXCsWJFNCHFHLKWrnBNZ66c2m",
		    "transactions": [],
		    "block_extensions": [],
		    "id": "0137c067c65e9db8f8ee467c856fb6d1779dfeb0332a971754156d075c9a37ca",
		    "block_num": 20430951,
		    "ref_block_prefix": 2085023480
		}
	*/
	openwallet.BlockHeader
	transactions []eos.TransactionReceipt
	Height       uint32 `storm:"id"`
	Fork         bool
}

//UnscanRecord 扫描失败的区块及交易
type UnscanRecord struct {
	ID          string `storm:"id"` // primary key
	BlockHeight uint64
	TxID        string
	Reason      string
}

//NewUnscanRecord new UnscanRecord
func NewUnscanRecord(height uint64, txID, reason string) *UnscanRecord {
	obj := UnscanRecord{}
	obj.BlockHeight = height
	obj.TxID = txID
	obj.Reason = reason
	obj.ID = common.Bytes2Hex(crypto.SHA256([]byte(fmt.Sprintf("%d_%s", height, txID))))
	return &obj
}

// TransferAction transfer action
type TransferAction struct {
	*eos.Action
	TransferData
}

// TransferData token contract transfer action data
type TransferData struct {
	From     string    `json:"from,omitempty"`
	To       string    `json:"to,omitempty"`
	Quantity eos.Asset `json:"quantity,omitempty"`
	Memo     string    `json:"memo,omitempty"`
}

type ParamData struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Amount eos.Asset `json:"amount"`
	Memo   string    `json:"memo"`
}

//type Operations struct {
//	OperationsSub []interface{} `json:"operations_sub"`
//}

type TransConMainTest struct {
	RefBlockNum    uint16          `json:"ref_block_num"`
	RefBlockPrefix uint32          `json:"ref_block_prefix"`
	Expiration     eos.JSONTime    `json:"expiration"`
	Operations     [][]interface{} `json:"operations"`
	Extensions     []interface{}   `json:"extensions"`
	Signatures     []ecc.Signature `json:"signatures"`
}

type ParamDataTest struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Amount eos.Asset `json:"amount"`
	Memo   string    `json:"memo"`
}
type TransConMain struct {
	RefBlockNum    uint16         `json:"ref_block_num"`
	RefBlockPrefix uint32         `json:"ref_block_prefix"`
	Expiration     eos.JSONTime   `json:"expiration"`
	Operations     [][]*ParamData `json:"operations"`
	Extensions     []interface{}  `json:"extensions"`
	Signatures     []string       `json:"signatures"`
}

type TransConMainSub struct {
	RefBlockNum    uint16          `json:"ref_block_num"`
	RefBlockPrefix uint32          `json:"ref_block_prefix"`
	Expiration     eos.JSONTime    `json:"expiration"`
	Operations     [][]interface{} `json:"operations"`
	Extensions     []interface{}   `json:"extensions"`
	Signatures     []string        `json:"signatures"`
}

//type Operations struct {
//}

// ParseHeader 区块链头
func ParseHeader(b *ApiBlock,symbol string) *openwallet.BlockHeader {
	obj := openwallet.BlockHeader{}

	//解析josn
	//obj.Merkleroot = b.TransactionMRoot.String()
	obj.Hash = b.Hash
	obj.Previousblockhash = b.PreviousHash
	obj.Height = uint64(b.Height)
	obj.Time = uint64(b.Timestamp)
	obj.Symbol = symbol
	return &obj
}

// ParseBlock 区块
func ParseBlock(b *ApiBlock,symbol string) *Block {
	obj := Block{}

	//解析josn
	//obj.Merkleroot = b.TransactionMRoot.String()
	obj.Hash = b.Hash
	obj.Previousblockhash = b.PreviousHash
	obj.Height = uint32(b.Height)
	obj.Time = uint64(b.Timestamp)
	obj.Symbol = symbol
	return &obj
}
