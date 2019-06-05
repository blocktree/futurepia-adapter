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
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"time"

	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
)

const (
	blockchainBucket = "blockchain" // blockchain dataset
	//periodOfTask      = 5 * time.Second // task interval
	maxExtractingSize = 10 // thread count
	coinTag           = "PIA"
)

//PIABlockScanner PIA block scanner
type PIABlockScanner struct {
	*openwallet.BlockScannerBase

	CurrentBlockHeight   uint64         //当前区块高度
	extractingCH         chan struct{}  //扫描工作令牌
	wm                   *WalletManager //钱包管理者
	IsScanMemPool        bool           //是否扫描交易池
	RescanLastBlockCount uint64         //重扫上N个区块数量
}

//ExtractResult extract result
type ExtractResult struct {
	extractData map[string][]*openwallet.TxExtractData
	TxID        string
	BlockHash   string
	BlockHeight uint64
	BlockTime   int64
	Success     bool
}

//SaveResult result
type SaveResult struct {
	TxID        string
	BlockHeight uint64
	Success     bool
}

// NewPIABlockScanner create a block scanner
func NewPIABlockScanner(wm *WalletManager) *PIABlockScanner {
	bs := PIABlockScanner{
		BlockScannerBase: openwallet.NewBlockScannerBase(),
	}

	bs.extractingCH = make(chan struct{}, maxExtractingSize)
	bs.wm = wm
	bs.IsScanMemPool = true
	bs.RescanLastBlockCount = 0

	// set task
	bs.SetTask(bs.ScanBlockTask)

	return &bs
}

// ScanBlockTask scan block task
func (bs *PIABlockScanner) ScanBlockTask() {

	var (
		currentHeight uint32
		currentHash   string
	)

	// get local block header
	currentHeight, currentHash, err := bs.GetLocalBlockHead()

	if err != nil {
		bs.wm.Log.Std.Error("", err)
	}

	if currentHeight == 0 {
		bs.wm.Log.Std.Info("No records found in local, get current block as the local!")

		headBlock, err := bs.GetGlobalHeadBlock()
		if err != nil {
			bs.wm.Log.Std.Info("get head block error, err=%v", err)
		}

		currentHash = headBlock.PreviousHash
		currentHeight = uint32(headBlock.Height - 1)
	}

	for {
		if !bs.Scanning {
			// stop scan
			return
		}

		infoResp, err := bs.GetGlobalHeadBlock()
		if err != nil {
			bs.wm.Log.Errorf("GetGlobalHeadBlock failed, err=%v", err)
			break
		}

		maxBlockHeight := uint32(infoResp.Height)

		bs.wm.Log.Info("current block height:", currentHeight, " maxBlockHeight:", maxBlockHeight)
		if currentHeight == maxBlockHeight {
			bs.wm.Log.Std.Info("block scanner has scanned full chain data. Current height %d", maxBlockHeight)
			break
		}

		// next block
		currentHeight = currentHeight + 1

		bs.wm.Log.Std.Info("block scanner scanning height: %d ...", currentHeight)
		block, err := bs.wm.Api.getGetBlock(uint64(currentHeight))

		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data by rpc; unexpected error: %v", err)
			break
		}
		hash := block.Hash

		if currentHash != block.PreviousHash {
			bs.wm.Log.Std.Info("block has been fork on height: %d.", currentHeight)
			bs.wm.Log.Std.Info("block height: %d local hash = %s ", currentHeight-1, currentHash)
			bs.wm.Log.Std.Info("block height: %d mainnet hash = %s ", currentHeight-1, block.PreviousHash)
			bs.wm.Log.Std.Info("delete recharge records on block height: %d.", currentHeight-1)

			// get local fork bolck
			forkBlock, _ := bs.GetLocalBlock(currentHeight - 1)
			// delete last unscan block
			bs.DeleteUnscanRecord(currentHeight - 1)
			currentHeight = currentHeight - 2 // scan back to last 2 block
			if currentHeight <= 0 {
				currentHeight = 1
			}
			localBlock, err := bs.GetLocalBlock(currentHeight)
			if err != nil {
				bs.wm.Log.Std.Error("block scanner can not get local block; unexpected error: %v", err)
				//get block from rpc
				bs.wm.Log.Info("block scanner prev block height:", currentHeight)
				curBlock, err := bs.wm.Api.getGetBlock(uint64(currentHeight))
				if err != nil {
					bs.wm.Log.Std.Error("block scanner can not get prev block by rpc; unexpected error: %v", err)
					break
				}
				currentHash = curBlock.Hash
			} else {
				//重置当前区块的hash
				currentHash = localBlock.Hash
			}
			bs.wm.Log.Std.Info("rescan block on height: %d, hash: %s .", currentHeight, currentHash)

			//重新记录一个新扫描起点
			bs.SaveLocalBlockHead(currentHeight, currentHash)

			if forkBlock != nil {
				//通知分叉区块给观测者，异步处理
				bs.forkBlockNotify(forkBlock)
			}

		} else {
			err := bs.BatchExtractTransactions(uint64(currentHeight), currentHash, block.Timestamp, block.LocalTransactions)
			if err != nil {
				bs.wm.Log.Std.Error("block scanner ran BatchExtractTransactions occured unexpected error: %v", err)
			}

			//重置当前区块的hash
			currentHash = hash
			//保存本地新高度
			bs.SaveLocalBlockHead(currentHeight, currentHash)
			bs.SaveLocalBlock(ParseBlock(block))
			//通知新区块给观测者，异步处理
			bs.newBlockNotify(block)
		}
	}
}

//newBlockNotify 获得新区块后，通知给观测者
func (bs *PIABlockScanner) forkBlockNotify(block *Block) {
	header := block.BlockHeader
	header.Fork = true
	bs.NewBlockNotify(&header)
}

//newBlockNotify 获得新区块后，通知给观测者
func (bs *PIABlockScanner) newBlockNotify(block *ApiBlock) {
	header := ParseHeader(block)
	bs.NewBlockNotify(header)
}

// BatchExtractTransactions 批量提取交易单
func (bs *PIABlockScanner) BatchExtractTransactions(blockHeight uint64, blockHash string, blockTime int64, transactions []*LocalTransaction) error {

	var (
		quit       = make(chan struct{})
		done       = 0 //完成标记
		failed     = 0
		shouldDone = len(transactions) //需要完成的总数
	)

	if len(transactions) == 0 {
		return nil
	}

	bs.wm.Log.Std.Info("block scanner ready extract transactions total: %d ", len(transactions))

	//生产通道
	producer := make(chan ExtractResult)
	defer close(producer)

	//消费通道
	worker := make(chan ExtractResult)
	defer close(worker)

	//保存工作
	saveWork := func(height uint64, result chan ExtractResult) {
		//回收创建的地址
		for gets := range result {

			if gets.Success {
				notifyErr := bs.newExtractDataNotify(height, gets.extractData)
				if notifyErr != nil {
					failed++ //标记保存失败数
					bs.wm.Log.Std.Info("newExtractDataNotify unexpected error: %v", notifyErr)
				}
			} else {
				//记录未扫区块
				unscanRecord := NewUnscanRecord(height, "", "")
				bs.SaveUnscanRecord(unscanRecord)
				failed++ //标记保存失败数
			}
			//累计完成的线程数
			done++
			if done == shouldDone {
				close(quit) //关闭通道，等于给通道传入nil
			}
		}
	}

	//提取工作
	extractWork := func(eblockHeight uint64, eBlockHash string, eBlockTime int64, mTransactions []*LocalTransaction, eProducer chan ExtractResult) {
		for _, tx := range mTransactions {
			bs.extractingCH <- struct{}{}
			go func(mBlockHeight uint64, mTx *LocalTransaction, end chan struct{}, mProducer chan<- ExtractResult) {
				//导出提出的交易
				mProducer <- bs.ExtractTransaction(mBlockHeight, eBlockHash, eBlockTime, mTx, bs.ScanTargetFunc)
				//释放
				<-end

			}(eblockHeight, tx, bs.extractingCH, eProducer)
		}
	}
	/*	开启导出的线程	*/

	//独立线程运行消费
	go saveWork(blockHeight, worker)

	//独立线程运行生产
	go extractWork(blockHeight, blockHash, blockTime, transactions, producer)

	//以下使用生产消费模式
	bs.extractRuntime(producer, worker, quit)

	if failed > 0 {
		return fmt.Errorf("block scanner saveWork failed")
	}

	return nil
}

//extractRuntime 提取运行时
func (bs *PIABlockScanner) extractRuntime(producer chan ExtractResult, worker chan ExtractResult, quit chan struct{}) {

	var (
		values = make([]ExtractResult, 0)
	)

	for {
		var activeWorker chan<- ExtractResult
		var activeValue ExtractResult
		//当数据队列有数据时，释放顶部，传输给消费者
		if len(values) > 0 {
			activeWorker = worker
			activeValue = values[0]
		}
		select {
		//生成者不断生成数据，插入到数据队列尾部
		case pa := <-producer:
			values = append(values, pa)
		case <-quit:
			//退出
			return
		case activeWorker <- activeValue:
			values = values[1:]
		}
	}
	//return
}

// ExtractTransaction 提取交易单
func (bs *PIABlockScanner) ExtractTransaction(blockHeight uint64, blockHash string, blockTime int64, transaction *LocalTransaction, scanTargetFunc openwallet.BlockScanTargetFunc) ExtractResult {
	var (
		success = true
		result  = ExtractResult{
			BlockHash:   blockHash,
			BlockHeight: blockHeight,
			TxID:        transaction.TxId,
			extractData: make(map[string][]*openwallet.TxExtractData),
			BlockTime:   blockTime,
		}
	)

	if transaction.CoinTag != coinTag {
		bs.wm.Log.Std.Debug("transaction not the pia: %s", transaction.TxId, ":", transaction.CoinTag)
		return ExtractResult{Success: true}
	}

	//提出交易单明细
	if transaction.TxId == "" {
		bs.wm.Log.Std.Debug("transaction packed empty: %s", transaction.TxId)
		return ExtractResult{Success: true}
	}

	if transaction.Type == "transfer" {

		if scanTargetFunc == nil {
			bs.wm.Log.Std.Error("scanTargetFunc is not configurated")
			return ExtractResult{Success: false}
		}

		//订阅地址为交易单中的发送者
		accountID1, ok1 := scanTargetFunc(openwallet.ScanTarget{Alias: transaction.From, Symbol: bs.wm.Symbol(), BalanceModelType: openwallet.BalanceModelTypeAccount})
		//订阅地址为交易单中的接收者
		accountID2, ok2 := scanTargetFunc(openwallet.ScanTarget{Alias: transaction.To, Symbol: bs.wm.Symbol(), BalanceModelType: openwallet.BalanceModelTypeAccount})
		if accountID1 == accountID2 && len(accountID1) > 0 && len(accountID2) > 0 {
			bs.InitExtractResult(accountID1,transaction, &result, 0)
		} else {
			if ok1 {
				bs.InitExtractResult(accountID1,transaction, &result, 1)
			}

			if ok2 {
				bs.InitExtractResult(accountID2, transaction, &result, 2)
			}
		}

	}
	result.Success = success
	return result

}

//InitExtractResult optType = 0: 输入输出提取，1: 输入提取，2：输出提取
func (bs *PIABlockScanner) InitExtractResult(sourceKey string, localTransaction *LocalTransaction, result *ExtractResult, optType int64) {


	txExtractDataArray := result.extractData[sourceKey]
	if txExtractDataArray == nil {
		txExtractDataArray = make([]*openwallet.TxExtractData, 0)
	}

	txExtractData := &openwallet.TxExtractData{}

	status := "1"
	reason := ""

	symbol := bs.wm.Symbol()
	decimals := bs.wm.Decimal()
	amount := localTransaction.Amount

	/** 不把他当成代币了
	contractID := openwallet.GenContractID(bs.wm.Symbol(), string(action.Account))
	coin.Contract = openwallet.SmartContract{
		Symbol:     bs.wm.Symbol(),
		ContractID: contractID,
		Address:    string(action.Account),
		Token:      symbol,
	}
	*/

	coin := openwallet.Coin{
		Symbol:     symbol,
		IsContract: false,
		//ContractID: contractID,
	}
	//


	transx := &openwallet.Transaction{
		Fees:        "0",
		Coin:        coin,
		BlockHash:   result.BlockHash,
		BlockHeight: result.BlockHeight,
		TxID:        result.TxID,
		Decimal:     decimals,
		Amount:      amount,
		ConfirmTime: result.BlockTime,
		From:        []string{localTransaction.From + ":" + amount},
		To:          []string{localTransaction.To + ":" + amount},
		IsMemo:      true,
		Status:      status,
		Reason:      reason,
	}

	transx.SetExtParam("memo", localTransaction.Memo)

	wxID := openwallet.GenTransactionWxID(transx)
	transx.WxID = wxID

	txExtractData.Transaction = transx
	if optType == 0 {
		bs.extractTxInput(localTransaction, txExtractData)
		bs.extractTxOutput(localTransaction, txExtractData)
	} else if optType == 1 {
		bs.extractTxInput(localTransaction, txExtractData)
	} else if optType == 2 {
		bs.extractTxOutput(localTransaction, txExtractData)
	}

	txExtractDataArray = append(txExtractDataArray, txExtractData)
	result.extractData[sourceKey] = txExtractDataArray
}

//extractTxInput 提取交易单输入部分,无需手续费，所以只包含1个TxInput
func (bs *PIABlockScanner) extractTxInput(transaction *LocalTransaction, txExtractData *openwallet.TxExtractData) {

	tx := txExtractData.Transaction
	coin := openwallet.Coin(tx.Coin)

	//主网from交易转账信息，第一个TxInput
	txInput := &openwallet.TxInput{}
	txInput.Recharge.Sid = openwallet.GenTxInputSID(tx.TxID, bs.wm.Symbol(), coin.ContractID, uint64(0))
	txInput.Recharge.TxID = tx.TxID
	txInput.Recharge.Address = transaction.From
	txInput.Recharge.Coin = coin
	txInput.Recharge.Amount = tx.Amount
	txInput.Recharge.Symbol = coin.Symbol
	//txInput.Recharge.IsMemo = true
	//txInput.Recharge.Memo = data.Memo
	txInput.Recharge.BlockHash = tx.BlockHash
	txInput.Recharge.BlockHeight = tx.BlockHeight
	txInput.Recharge.Index = 0 //账户模型填0
	txInput.Recharge.CreateAt = time.Now().Unix()
	txExtractData.TxInputs = append(txExtractData.TxInputs, txInput)
}

//extractTxOutput 提取交易单输入部分,只有一个TxOutPut
func (bs *PIABlockScanner) extractTxOutput(transaction *LocalTransaction, txExtractData *openwallet.TxExtractData) {

	tx := txExtractData.Transaction
	//data := transaction
	coin := openwallet.Coin(tx.Coin)

	//主网to交易转账信息,只有一个TxOutPut
	txOutput := &openwallet.TxOutPut{}
	txOutput.Recharge.Sid = openwallet.GenTxOutPutSID(tx.TxID, bs.wm.Symbol(), coin.ContractID, uint64(0))
	txOutput.Recharge.TxID = tx.TxID
	txOutput.Recharge.Address = transaction.To
	txOutput.Recharge.Coin = coin
	txOutput.Recharge.Amount = tx.Amount
	txOutput.Recharge.Symbol = coin.Symbol
	//txOutput.Recharge.IsMemo = true
	//txOutput.Recharge.Memo = data.Memo
	txOutput.Recharge.BlockHash = tx.BlockHash
	txOutput.Recharge.BlockHeight = tx.BlockHeight
	txOutput.Recharge.Index = 0 //账户模型填0
	txOutput.Recharge.CreateAt = time.Now().Unix()
	txExtractData.TxOutputs = append(txExtractData.TxOutputs, txOutput)
}

//newExtractDataNotify 发送通知
func (bs *PIABlockScanner) newExtractDataNotify(height uint64, extractData map[string][]*openwallet.TxExtractData) error {
	for o := range bs.Observers {
		for key, array := range extractData {
			for _, item := range array {
				err := o.BlockExtractDataNotify(key, item)
				if err != nil {
					log.Error("BlockExtractDataNotify unexpected error:", err)
					//记录未扫区块
					unscanRecord := NewUnscanRecord(height, "", "ExtractData Notify failed.")
					err = bs.SaveUnscanRecord(unscanRecord)
					if err != nil {
						log.Std.Error("block height: %d, save unscan record failed. unexpected error: %v", height, err.Error())
					}

				}
			}

		}
	}

	return nil
}

//ScanBlock 扫描指定高度区块
func (bs *PIABlockScanner) ScanBlock(height uint64) error {

	block, err := bs.wm.Api.getGetBlock(uint64(height))
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

		//记录未扫区块
		unscanRecord := NewUnscanRecord(height, "", err.Error())
		bs.SaveUnscanRecord(unscanRecord)
		bs.wm.Log.Std.Info("block height: %d extract failed.", height)
		return err
	}

	bs.scanBlock(block)

	return nil
}

func (bs *PIABlockScanner) scanBlock(block *ApiBlock) error {

	bs.wm.Log.Std.Info("block scanner scanning height: %d ...", block.Hash)

	err := bs.BatchExtractTransactions(uint64(block.Height), block.Hash, block.Timestamp, block.LocalTransactions)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", err)
	}

	//通知新区块给观测者，异步处理
	bs.newBlockNotify(block)

	return nil
}

//SetRescanBlockHeight 重置区块链扫描高度
func (bs *PIABlockScanner) SetRescanBlockHeight(height uint64) error {
	if height <= 0 {
		return errors.New("block height to rescan must greater than 0. ")
	}

	block, err := bs.wm.Api.getGetBlock(uint64(height - 1))
	if err != nil {
		return err
	}

	bs.SaveLocalBlockHead(uint32(height-1), block.Hash)

	return nil
}

// GetGlobalMaxBlockHeight GetGlobalMaxBlockHeight
func (bs *PIABlockScanner) GetGlobalMaxBlockHeight() uint64 {
	headBlock, err := bs.wm.Api.getDynamicGlobal()
	if err != nil {
		bs.wm.Log.Std.Info("get global head block error;unexpected error:%v", err)
		return 0
	}
	return uint64(headBlock.Height)
}

//GetGlobalHeadBlock GetGlobalHeadBlock
func (bs *PIABlockScanner) GetGlobalHeadBlock() (block *ApiBlock, err error) {
	headBlock, err := bs.wm.Api.getDynamicGlobal()
	if err != nil {
		bs.wm.Log.Std.Info("get global head block error;unexpected error:%v", err)
		return
	}

	block, err = bs.wm.Api.getGetBlock(uint64(headBlock.Height - 1))
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get block by height; unexpected error:%v", err)
		return
	}

	return
}

//GetCurrentBlockHeader 获取当前区块高度
func (bs *PIABlockScanner) GetCurrentBlockHeader() (*openwallet.BlockHeader, error) {

	apiBlock ,err := bs.GetGlobalHeadBlock()
	if err != nil {
		return nil,err
	}

	return &openwallet.BlockHeader{Height: uint64(apiBlock.Height), Hash: string(apiBlock.Hash)}, nil
}

//GetScannedBlockHeader 获取当前扫描的区块头
func (bs *PIABlockScanner) GetScannedBlockHeader() (*openwallet.BlockHeader, error) {

	var (
		blockHeader *openwallet.BlockHeader
		blockHeight uint64 = 0
		hash        string
		err         error
	)

	head, err := bs.GetGlobalHeadBlock()
	blockHeight = uint64(head.Height)
	//如果本地没有记录，查询接口的高度
	if blockHeight == 0 {
		blockHeader, err = bs.GetCurrentBlockHeader()
		if err != nil {

			return nil, err
		}
		blockHeight = blockHeader.Height
		//就上一个区块链为当前区块
		blockHeight = blockHeight - 1
		curBlock, err := bs.wm.Api.getGetBlock(uint64(blockHeight))
		if err != nil {
			return nil, err
		}
		hash = curBlock.Hash
	}

	return &openwallet.BlockHeader{Height: blockHeight, Hash: hash}, nil
}


//GetScannedBlockHeight 获取已扫区块高度
func (bs *PIABlockScanner) GetScannedBlockHeight() uint64 {
	height, _, _ := bs.GetLocalBlockHead()
	return uint64(height)
}

//GetBalanceByAddress 查询地址余额
func (bs *PIABlockScanner) GetBalanceByAddress(address ...string) ([]*openwallet.Balance, error) {

	addrBalanceArr := make([]*openwallet.Balance, 0)
	for _, a := range address {
		acc, err := bs.wm.Api.GetBalance(a)
		if err == nil {

			b,_ := decimal.NewFromString(acc.Balance)
			obj := &openwallet.Balance{
				Symbol:           bs.wm.Symbol(),
				Address:          a,
				Balance:          b.String(),
				UnconfirmBalance: b.String(),
				ConfirmBalance:   b.String(),
			}

			addrBalanceArr = append(addrBalanceArr, obj)
		}

	}

	return addrBalanceArr, nil
}
