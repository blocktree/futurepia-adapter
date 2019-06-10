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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/blocktree/futurepia-adapter/futurepia_txsigner"
	"github.com/blocktree/go-owcrypt"
	"github.com/eoscanada/eos-go"
	"github.com/pkg/errors"
	"time"

	"github.com/blocktree/openwallet/openwallet"
	"github.com/shopspring/decimal"
)

// TransactionDecoder 交易单解析器
type TransactionDecoder struct {
	openwallet.TransactionDecoderBase
	wm *WalletManager //钱包管理者
}

//NewTransactionDecoder 交易单解析器
func NewTransactionDecoder(wm *WalletManager) *TransactionDecoder {
	decoder := TransactionDecoder{}
	decoder.wm = wm
	return &decoder
}

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		accountID      = rawTx.Account.AccountID
		accountBalance *ApiBalance
		amountStr      string
		to             string
	)

	//codeAccount := rawTx.Coin.Contract.Address

	//获取wallet
	account, err := wrapper.GetAssetsAccountInfo(accountID)
	if err != nil {
		return err
	}

	if account.Alias == "" {
		return fmt.Errorf("[%s] have not been created", accountID)
	}

	//账户是否上链
	accountBalance, err = decoder.wm.Api.GetBalance(account.Alias)
	if err != nil || accountBalance == nil {
		return fmt.Errorf("pia account of from not found on chain")
	}

	for k, v := range rawTx.To {
		amountStr = v
		to = k
		break
	}

	// 检查目标账户是否存在
	accountTo, err := decoder.wm.Api.GetBalance(to)
	if err != nil || accountTo == nil {
		return fmt.Errorf("pia account of to not found on chain")
	}

	//accountBalanceDec := decimal.New(int64(accountBalance.Amount), -int32(accountBalance.Precision))
	accountBalanceDec, _ := decimal.NewFromString(accountBalance.Balance)
	amountDec, _ := decimal.NewFromString(amountStr)

	if accountBalanceDec.LessThan(amountDec) {
		return fmt.Errorf("the balance: %s is not enough", amountStr)
	}

	memo := rawTx.GetExtParam().Get("memo").String()

	createTxErr := decoder.createRawTransaction(
		wrapper,
		rawTx,
		account.Alias,
		amountDec,
		to,
		memo)
	if createTxErr != nil {
		return createTxErr
	}

	return nil

}

//SignRawTransaction 签名交易单
func (decoder *TransactionDecoder) SignRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		//this.wm.Log.Std.Error("len of signatures error. ")
		return fmt.Errorf("transaction signature is empty")
	}

	key, err := wrapper.HDKey()
	if err != nil {
		return err
	}

	keySignatures := rawTx.Signatures[rawTx.Account.AccountID]
	if keySignatures != nil {
		for _, keySignature := range keySignatures {

			childKey, err := key.DerivedKeyWithPath(keySignature.Address.HDPath, keySignature.EccType)
			keyBytes, err := childKey.GetPrivateKeyBytes()
			if err != nil {
				return err
			}
			//decoder.wm.Log.Debug("privateKey:", hex.EncodeToString(keyBytes))

			hash, err := hex.DecodeString(keySignature.Message)
			if err != nil {
				return fmt.Errorf("decoder transaction hash failed, unexpected err: %v", err)
			}

			//decoder.wm.Log.Debug("hash:", hash)

			sig, err := futurepia_txsigner.Default.SignTransactionHash(hash, keyBytes, decoder.wm.CurveType())
			if err != nil {
				return fmt.Errorf("sign transaction hash failed, unexpected err: %v", err)
			}

			keySignature.Signature = hex.EncodeToString(sig)
		}
	}

	decoder.wm.Log.Info("transaction hash sign success")

	rawTx.Signatures[rawTx.Account.AccountID] = keySignatures

	return nil
}

//VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *TransactionDecoder) VerifyRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		//this.wm.Log.Std.Error("len of signatures error. ")
		return fmt.Errorf("transaction signature is empty")
	}

	var tx *TransConMain
	txHex, err := hex.DecodeString(rawTx.RawHex)
	if err != nil {
		return fmt.Errorf("transaction decode failed, unexpected error: %v", err)
	}
	err = eos.UnmarshalBinary(txHex, &tx)
	if err != nil {
		return fmt.Errorf("transaction decode failed, unexpected error: %v", err)
	}

	//支持多重签名
	for accountID, keySignatures := range rawTx.Signatures {
		decoder.wm.Log.Debug("accountID Signatures:", accountID)
		for _, keySignature := range keySignatures {

			messsage, _ := hex.DecodeString(keySignature.Message)
			signature, _ := hex.DecodeString(keySignature.Signature)
			publicKey, _ := hex.DecodeString(keySignature.Address.PublicKey)

			//验证签名
			uncompessedPublicKey := owcrypt.PointDecompress(publicKey, decoder.wm.CurveType())
			//decoder.wm.Log.Debugf("publicKey: %s", hex.EncodeToString(uncompessedPublicKey))
			valid, compactSig, err := futurepia_txsigner.Default.VerifyAndCombineSignature(messsage, uncompessedPublicKey[1:], signature)
			if !valid {
				return fmt.Errorf("transaction verify failed: %v", err)
			}

			tx.Signatures = append(
				tx.Signatures,
				hex.EncodeToString(compactSig),
			)
		}
	}

	bin, err := eos.MarshalBinary(tx)
	if err != nil {
		return fmt.Errorf("signed transaction encode failed, unexpected error: %v", err)
	}

	rawTx.IsCompleted = true
	rawTx.RawHex = hex.EncodeToString(bin)

	return nil
}

// SubmitRawTransaction 广播交易单
func (decoder *TransactionDecoder) SubmitRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) (*openwallet.Transaction, error) {

	var stx *TransConMain
	txHex, err := hex.DecodeString(rawTx.RawHex)
	if err != nil {
		return nil, fmt.Errorf("transaction decode failed, unexpected error: %v", err)
	}
	err = eos.UnmarshalBinary(txHex, &stx)
	if err != nil {
		return nil, fmt.Errorf("transaction decode failed, unexpected error: %v", err)
	}

	var params = make([]interface{}, 0)
	params = append(params, "transfer")
	params = append(params, stx.Operations[0][0])
	var paramss = make([][]interface{}, 0)
	paramss = append(paramss, params)

	tranSub := &TransConMainSub{
		RefBlockNum:    stx.RefBlockNum,
		RefBlockPrefix: stx.RefBlockPrefix,
		Expiration:     stx.Expiration,
		Signatures:     stx.Signatures,
		Extensions:     stx.Extensions,
		Operations:     paramss,
	}
	resultee, err := decoder.wm.Api.PushTransaction(tranSub)
	if err != nil {
		return nil, fmt.Errorf("push transaction: %s", err)
	}

	//log.Warn(resultee)
	//log.Infof("Transaction [%s] submitted to the network successfully.", hex.EncodeToString(response.Processed.ID))

	rawTx.TxID = resultee.Id
	rawTx.IsSubmit = true

	decimals := int32(rawTx.Coin.Contract.Decimals)
	fees := "0"

	//记录一个交易单
	tx := &openwallet.Transaction{
		From:       rawTx.TxFrom,
		To:         rawTx.TxTo,
		Amount:     rawTx.TxAmount,
		Coin:       rawTx.Coin,
		TxID:       rawTx.TxID,
		Decimal:    decimals,
		AccountID:  rawTx.Account.AccountID,
		Fees:       fees,
		SubmitTime: time.Now().Unix(),
		ExtParam:   rawTx.ExtParam,
	}

	tx.WxID = openwallet.GenTransactionWxID(tx)

	return tx, nil
}

//GetRawTransactionFeeRate 获取交易单的费率
func (decoder *TransactionDecoder) GetRawTransactionFeeRate() (feeRate string, unit string, err error) {
	return "0", "pia", nil
}

//CreateSummaryRawTransaction 创建汇总交易
func (decoder *TransactionDecoder) CreateSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {
	var (
		rawTxWithErrArray []*openwallet.RawTransactionWithError
		rawTxArray        = make([]*openwallet.RawTransaction, 0)
		err               error
	)
	rawTxWithErrArray, err = decoder.CreateSummaryRawTransactionWithError(wrapper, sumRawTx)
	if err != nil {
		return nil, err
	}
	for _, rawTxWithErr := range rawTxWithErrArray {
		if rawTxWithErr.Error != nil {
			continue
		}
		rawTxArray = append(rawTxArray, rawTxWithErr.RawTx)
	}
	return rawTxArray, nil
}

//CreateSummaryRawTransactionWithError 创建汇总交易
func (decoder *TransactionDecoder) CreateSummaryRawTransactionWithError(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {

	var (
		rawTxArray = make([]*openwallet.RawTransactionWithError, 0)
		accountID  = sumRawTx.Account.AccountID
	)

	minTransfer, _ := decimal.NewFromString(sumRawTx.MinTransfer)
	retainedBalance, _ := decimal.NewFromString(sumRawTx.RetainedBalance)

	//codeAccount := sumRawTx.Coin.Contract.Address
	//tokenCoin := sumRawTx.Coin.Contract.Token
	//tokenDecimals := rawTx.Coin.Contract.Decimals

	if minTransfer.LessThan(retainedBalance) {
		return nil, fmt.Errorf("mini transfer amount must be greater than address retained balance")
	}

	//获取wallet
	account, err := wrapper.GetAssetsAccountInfo(accountID)
	if err != nil {
		return nil, err
	}

	if account.Alias == "" {
		return nil, fmt.Errorf("[%s] have not been created", accountID)
	}

	accountAsset, err := decoder.wm.Api.GetBalance(account.Alias)
	if err != nil {
		return nil, fmt.Errorf("pia account of to not found on chain")
	}
	accountBalanceDec, err := decimal.NewFromString(accountAsset.Balance)
	if err != nil {
		return nil, fmt.Errorf("pia accountBalanceDec can't be decimal")
	}
	if accountBalanceDec.LessThan(minTransfer) || accountBalanceDec.LessThanOrEqual(decimal.Zero) {
		return rawTxArray, nil
	}

	//计算汇总数量 = 余额 - 保留余额
	sumAmount := accountBalanceDec.Sub(retainedBalance)

	quantity := sumAmount
	memo := sumRawTx.GetExtParam().Get("memo").String()
	decoder.wm.Log.Debugf("balance: %v", accountBalanceDec.String())
	decoder.wm.Log.Debugf("fees: %d", 0)
	decoder.wm.Log.Debugf("sumAmount: %v", sumAmount)

	//创建一笔交易单
	rawTx := &openwallet.RawTransaction{
		Coin:    sumRawTx.Coin,
		Account: sumRawTx.Account,
		To: map[string]string{
			sumRawTx.SummaryAddress: sumAmount.String(),
		},
		Required: 1,
	}

	createTxErr := decoder.createRawTransaction(
		wrapper,
		rawTx,
		account.Alias,
		quantity,
		sumRawTx.SummaryAddress,
		memo)
	rawTxWithErr := &openwallet.RawTransactionWithError{
		RawTx: rawTx,
		Error: createTxErr,
	}

	//创建成功，添加到队列
	rawTxArray = append(rawTxArray, rawTxWithErr)

	return rawTxArray, nil
}

//createRawTransaction
func (decoder *TransactionDecoder) createRawTransaction(
	wrapper openwallet.WalletDAI,
	rawTx *openwallet.RawTransaction,
	accountName string,
	quantity decimal.Decimal,
	to string,
	memo string) *openwallet.Error {

	apiHead, err := decoder.wm.Api.getDynamicGlobal()
	if err != nil {
		return openwallet.NewError(3004, "createRawTransaction-getDynamicGlobal err :"+err.Error())
	}

	apiBlock, err := decoder.wm.Api.getGetBlock(uint64(apiHead.LastIrreversible))
	if err != nil {
		return openwallet.NewError(3004, "createRawTransaction-getGetBlock err :"+err.Error())
	}

	var (
		accountTotalSent = decimal.Zero
		txFrom           = make([]string, 0)
		txTo             = make([]string, 0)
		keySignList      = make([]*openwallet.KeySignature, 0)
		accountID        = rawTx.Account.AccountID
		amountDec        = decimal.Zero
	)

	for k, v := range rawTx.To {
		to = k
		amountDec, _ = decimal.NewFromString(v)
		break
	}

	for k, v := range rawTx.To {
		to = k
		amountDec, _ = decimal.NewFromString(v)
		break
	}
	var params = make([]*ParamData, 0)
	params = append(params, &ParamData{
		From: accountName,
		To:   to,
		Memo: memo,
		Amount: eos.Asset{
			Amount: eos.Int64(amountDec.Shift(int32(decoder.wm.Decimal())).IntPart()),
			Symbol: eos.Symbol{
				Precision: uint8(decoder.wm.Decimal()),
				Symbol:    decoder.wm.Symbol(),
			},
		},
	})
	var paramss = make([][]*ParamData, 0)
	paramss = append(paramss, params)
	now := time.Now()
	mm, _ := time.ParseDuration("30m")
	extensionTime := now.Add(mm)

	loc := time.FixedZone("GMT", 0)

	realTime := extensionTime.In(loc).Format(eos.JSONTimeFormat)
	eosTime, err := eos.ParseJSONTime(realTime)
	if err != nil {
		return openwallet.NewError(3001, "createRawTransaction-ParseJSONTime err :"+err.Error())
	}
	transCoin := &TransConMain{
		RefBlockNum:    apiHead.GetRefBlockNum(),
		RefBlockPrefix: apiBlock.GetRefBlockPrefix(),
		Expiration:     eosTime,
		Operations:     paramss,
	}

	txdata, err := eos.MarshalBinary(transCoin)
	if err != nil {
		return openwallet.ConvertError(errors.New(" MarshalBinary err :" + err.Error()))
	}
	txdataCopy := make([]byte, 0)

	for _, v := range txdata {
		txdataCopy = append(txdataCopy, v)
	}
	txdata[11] = 2
	txdata = txdata[:len(txdata)-1]
	addresses, err := wrapper.GetAddressList(0, -1,
		"AccountID", accountID)
	if err != nil {
		return openwallet.ConvertError(err)
	}

	chainId, err := hex.DecodeString(decoder.wm.Config.ChainId)

	//交易哈希
	//sigDigest := eos.SigDigest(chainId, txdata, nil)

	sigDigest := make([]byte, 0)
	for _, v := range chainId {
		sigDigest = append(sigDigest, v)
	}
	for _, v := range txdata {
		sigDigest = append(sigDigest, v)
	}
	if len(addresses) == 0 {
		return openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "[%s] have not PIA public key", accountID)
	}
	sig256 := sha256.Sum256(sigDigest)
	sigDigest2 := make([]byte, 0)
	for _, v := range sig256 {
		sigDigest2 = append(sigDigest2, v)
	}
	for _, addr := range addresses {
		signature := openwallet.KeySignature{
			EccType: decoder.wm.Config.CurveType,
			Nonce:   "",
			Address: addr,
			Message: hex.EncodeToString(sigDigest2),
		}
		keySignList = append(keySignList, &signature)
	}

	//计算账户的实际转账amount
	//accountTotalSentAddresses, findErr := wrapper.GetAddressList(0, -1, "AccountID", rawTx.Account.AccountID, "Address", to)
	if accountName != to {
		accountTotalSent = accountTotalSent.Add(amountDec)
	}
	accountTotalSent = decimal.Zero.Sub(accountTotalSent)

	txFrom = []string{fmt.Sprintf("%s:%s", accountName, amountDec.String())}
	txTo = []string{fmt.Sprintf("%s:%s", to, amountDec.String())}

	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	rawTx.RawHex = hex.EncodeToString(txdataCopy)
	rawTx.Signatures[rawTx.Account.AccountID] = keySignList
	rawTx.FeeRate = "0"
	rawTx.Fees = "0"
	rawTx.IsBuilt = true
	rawTx.TxAmount = accountTotalSent.String()
	rawTx.TxFrom = txFrom
	rawTx.TxTo = txTo

	return nil

}
