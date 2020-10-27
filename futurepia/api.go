/*
 * Copyright 2018 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */
package futurepia

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/blocktree/openwallet/v2/log"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

type Client struct {
	BaseURL   string
	Debug     bool
	ErrorTime int
	lock      sync.Mutex
	DelayTime int64
}

type Response struct {
	Id      int         `json:"id"`
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
}

type ApiHeadBlock struct {
	Height           int64  `json:"head_block_number"`
	Hash             string `json:"head_block_id"`
	LastIrreversible int64  `json:"last_irreversible_block_num"`
}

func (a *ApiHeadBlock) GetRefBlockNum() uint16 {
	return uint16((a.LastIrreversible - 1) & 0xFFFF)
}

type ApiBlock struct {
	Height            int64               `json:"block_number"`
	Hash              string              `json:"block_id"`
	PreviousHash      string              `json:"previous"`
	TimestampStr      string              `json:"timestamp"`
	Timestamp         int64               `json:"-"`
	Transactions      []*ApiTransaction   `json:"transactions"`
	TransactionIds    []string            `json:"transaction_ids"`
	LocalTransactions []*LocalTransaction `json:"-"`
}

func (a *ApiBlock) GetRefBlockPrefix() uint32 {
	result, _ := hex.DecodeString(a.PreviousHash)
	return readUInt32LE(result, 4, len(result))
}

func readUInt32LE(buf []byte, offset, byteLength int) uint32 {
	var n uint32
	buf = buf[offset : offset+byteLength]
	if len(buf) > 8 {
		buf = buf[:8]
	}
	for i, b := range buf {
		n += uint32(b) << uint(8*i)
	}
	return n
}

type ApiTransaction struct {
	Operations []interface{} `json:"operations"`
}

type ApiBalance struct {
	Name    string `json:"name"`
	Balance string `json:"balance"`
}

type LocalTransaction struct {
	From    string
	To      string
	Amount  string
	CoinTag string
	Memo    string
	Type    string
	TxId    string
	Index   int
}

//{"id":"8616b05f13cbedc1862435f18adfc89733c4025f","block_num":1565702,"trx_num":0,"expired":false}
type ApiTransResult struct {
	Id       string `json:"id"`
	BlockNum int64  `json:"block_num"`
	TrxNum   int    `json:"trx_num"`
	Expired  bool   `json:"expired"`
}

//获取最新高度区块信息
func (this *Client) GetDynamicGlobal() (*ApiHeadBlock, error) {
	params := []interface{}{
		//appendOxToAddress(addr),
		"database_api",
		"get_dynamic_global_properties",
		[]interface{}{},
	}
	result, err := this.Call("call", 1, params)
	if err != nil {
		log.Errorf("GetDynamicGlobal number faield, err = %v \n", err)
		return nil, err
	}

	if result.Type != gjson.JSON {
		log.Errorf("result of GetDynamicGlobal type error")
		return nil, errors.New("result of block number type error")
	}

	var apiHeadBlock *ApiHeadBlock
	err = json.Unmarshal([]byte(result.Raw), &apiHeadBlock)
	if err != nil {
		log.Errorf("decode json [%v] failed, err=%v", []byte(result.Raw), err)
		return nil, err
	}
	return apiHeadBlock, nil
}

//根据高度获取区块
func (this *Client) GetBalance(account ,feeString string) (*ApiBalance, error) {
	time.Sleep(200 * time.Millisecond)
	params := []interface{}{
		"database_api",
		"get_accounts",
		[]interface{}{[]interface{}{account}},
	}

	result, err := this.Call("call", 1, params)
	if err != nil {
		log.Errorf("get balance number faield,account = %s , err = %v \n", account, err)
		this.ErrorTime++
		if this.ErrorTime > 3 {
			this.ErrorTime = 0
			return nil, err
		} else {
			log.Errorf("reTry GetBalance")
			time.Sleep(2 * time.Second)
			return this.GetBalance(account,feeString)
		}
	}
	if result.Type != gjson.JSON {
		log.Errorf("result of GetBalance type error")
		return nil, errors.New("result of block number type error")
	}

	var apiBalances []*ApiBalance
	err = json.Unmarshal([]byte(result.Raw), &apiBalances)
	if err != nil {
		log.Errorf("GetBalance decode json [%v] failed, err=%v", []byte(result.Raw), err)
		return nil, err
	}

	if apiBalances == nil || len(apiBalances) == 0 {
		log.Errorf("GetBalance apiBalances is nil or length is 0", []byte(result.Raw))
		return nil, errors.New("GetBalance apiBalances is nil or length is 0")
	}
	balance := apiBalances[0]
	amountList := strings.Split(balance.Balance, " ")
	if len(amountList) != 2 {
		return nil, errors.New("GetBalance amountList is nil or length is not 2")
	}
	if amountList[1] != feeString {
		return nil, errors.New("GetBalance not PIA")
	}
	balance.Balance = amountList[0]
	return balance, nil
}

//获取最新高度区块
func (this *Client) getGetTopBlock() (*ApiBlock, error) {
	apiHead, err := this.GetDynamicGlobal()
	if err != nil {
		return nil, err
	}
	block, err := this.GetGetBlock(uint64(apiHead.Height))
	if err != nil {
		return nil, err
	}
	return block, nil
}

//根据高度获取区块
func (this *Client) GetGetBlock(block uint64) (*ApiBlock, error) {
	params := []interface{}{
		"database_api",
		"get_block",
		[]interface{}{block},
	}
	result, err := this.Call("call", 1, params)
	if err != nil {
		log.Errorf("get block number faield, err = %v \n", err)
		return nil, err
	}

	if result.Type != gjson.JSON {
		log.Errorf("result of block number type error")
		return nil, errors.New("result of block number type error")
	}

	var apiHeadBlock *ApiBlock
	err = json.Unmarshal([]byte(result.Raw), &apiHeadBlock)
	if err != nil {
		log.Errorf("decode json [%v] failed, err=%v", []byte(result.Raw), err)
		return nil, err
	}
	if apiHeadBlock != nil && apiHeadBlock.Height == 0 {
		apiHeadBlock.Height = int64(block)
	}
	//转换时间位unix

	loc := time.FixedZone("GMT", 0) //设置时区 GMT 0
	tt, _ := time.ParseInLocation("2006-01-02T15:04:05", apiHeadBlock.TimestampStr, loc)
	apiHeadBlock.Timestamp = tt.Unix()

	trans := apiHeadBlock.Transactions
	transactionIds := apiHeadBlock.TransactionIds
	apiHeadBlock.LocalTransactions = make([]*LocalTransaction, 0)
	if trans != nil && len(trans) > 0 && transactionIds != nil && len(transactionIds) > 0 {

		for index, tran := range trans {
			operations := tran.Operations
			if operations != nil && len(operations) > 0 {

				for i, opr := range operations {
					operationsTemp := opr.([]interface{})
					if operationsTemp == nil || len(operationsTemp) == 0 {
						continue
					}

					localTransaction := &LocalTransaction{}
					localTransaction.Type = operationsTemp[0].(string)
					mapTemp := operationsTemp[1].(map[string]interface{})
					if mapTemp != nil {
						main, ok := mapTemp["amount"]
						if !ok {
							continue
						}
						amountList := strings.Split(main.(string), " ")
						if len(amountList) != 2 {
							continue
						}
						if mapTemp["from"] == nil || mapTemp["to"] == nil {
							continue
						}

						localTransaction.CoinTag = amountList[1]
						localTransaction.Amount = amountList[0]
						localTransaction.From = mapTemp["from"].(string)
						localTransaction.To = mapTemp["to"].(string)
						localTransaction.Memo = mapTemp["memo"].(string)
						localTransaction.TxId = transactionIds[index]
						localTransaction.Index = i
					}
					apiHeadBlock.LocalTransactions = append(apiHeadBlock.LocalTransactions, localTransaction)
				}
			}
		}
	}
	return apiHeadBlock, nil
}

func (this *Client) PushTransaction(packedTx interface{}) (*ApiTransResult, error) {
	params := []interface{}{
		//appendOxToAddress(addr),
		"network_broadcast_api",
		"broadcast_transaction_synchronous",
		[]interface{}{packedTx},
	}
	result, err := this.Call("call", 1, params)
	if err != nil {
		log.Errorf("pushTransaction faield, err = %v \n", err)
		return nil, err
	}

	if result.Type != gjson.JSON {
		log.Errorf("result of pushTransaction type error")
		return nil, errors.New("result of pushTransaction type error")
	}

	var apiTransResult *ApiTransResult
	err = json.Unmarshal([]byte(result.Raw), &apiTransResult)
	if err != nil {
		log.Errorf("pushTransaction decode json [%v] failed, err=%v", []byte(result.Raw), err)
		return nil, err
	}

	return apiTransResult, nil
}

func (c *Client) Call(method string, id int64, params []interface{}) (*gjson.Result, error) {
	time.Sleep(time.Duration(c.DelayTime) * time.Millisecond)
	c.lock.Lock()
	defer c.lock.Unlock()
	authHeader := req.Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
	body := make(map[string]interface{}, 0)
	body["jsonrpc"] = "2.0"
	body["id"] = id
	body["method"] = method
	body["params"] = params

	if c.Debug {
		log.Debug("Start Request API...")
	}

	r, err := req.Post(c.BaseURL, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Debug("Request API Completed")
	}

	if c.Debug {
		log.Debugf("%+v\n", r)
	}

	if err != nil {
		return nil, err
	}

	resp := gjson.ParseBytes(r.Bytes())
	err = isError(&resp)
	if err != nil {
		return nil, err
	}

	result := resp.Get("result")

	return &result, nil
}

//isError 是否报错
func isError(result *gjson.Result) error {
	var (
		err error
	)

	if !result.Get("error").IsObject() {

		if !result.Get("result").Exists() {
			return errors.New("Response is empty! ")
		}

		return nil
	}

	errInfo := fmt.Sprintf("[%d]%s",
		result.Get("error.code").Int(),
		result.Get("error.message").String())
	err = errors.New(errInfo)

	return err
}
