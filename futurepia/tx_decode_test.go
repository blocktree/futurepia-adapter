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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/eoscanada/eos-go"
	"testing"
	"time"
)

func TestLastBlock(t *testing.T){
	v := 1562721
	result := (v-1) & 0xFFFF
	log.Warn(result)

	v1 := "0017d8d9794ec4b1a6e5181139af96965ebfac5e"
	result2 , _ := hex.DecodeString(v1)
	log.Warn(readUInt32LE(result2,4, len(result2)))
	//log.Warn(eos.JSONTime.Unix())
}



func TestTxBuild(t *testing.T) {
	var params = make([]interface{}, 0)
	//params = append(params,"")
	params = append(params, &ParamDataTest{
		From:   "kencani4",
		To:     "kencani",
		Memo:   "hello boy",
		Amount: eos.Asset{
			Amount:100000000,
			Symbol:eos.Symbol{
				Precision:8,
				Symbol:"PIA",
			},
		},
	})
	var paramss = make([][]interface{}, 0)
	paramss = append(paramss, params)



	//now := time.Now()
	now,_ := time.Parse("2006-01-02T15:04:05Z","2019-06-04T08:55:36.000Z")
	//mm, _ := time.ParseDuration("60m")
	//extensionTime := now.Add(mm)
	realTime := now.Format(eos.JSONTimeFormat)
	eosTime,err := eos.ParseJSONTime(realTime)
	if err != nil{
		//return openwallet.NewError(3001,"createRawTransaction-ParseJSONTime err :" + err.Error())
	}

	testMain := &TransConMainTest{
		RefBlockNum:    55513,
		RefBlockPrefix: 2982432377,
		Expiration:     eosTime,
		Operations:     paramss,
	}

	target := []byte{
		217,
		216,
		121,
		78,
		196,
		177,
		8,
		50,
		246,
		92,
		1,
		2,
		7,
		107,
		101,
		110,
		99,
		97,
		110,
		105,
		8,
		107,
		101,
		110,
		99,
		97,
		110,
		105,
		52,
		0,
		225,
		245,
		5,
		0,
		0,
		0,
		0,
		8,
		80,
		73,
		65,
		0,
		0,
		0,
		0,
		4,
		116,
		101,
		115,
		116,
		0,
	}
	//target = target[:len(target)-1]
	//log.Warn(target[:len(target)-1])
	//var buf bytes.Buffer
	//
	//enc := gob.NewEncoder(&buf)
	//
	//if err := enc.Encode(testMain); err != nil {
	//	log.Error("encode error:", err)
	//	return
	//}
	//fmt.Printf("% x\n", buf.Bytes())


	result2,_:=json.Marshal(testMain)
	log.Warn(string(result2))
	txdata, err := eos.MarshalBinary(testMain)
	txdata[11] = 2
	//testData := []byte(`[["transfer",{"from":"kencani","to":"kencani4","amount":"1.00000000 PIA","memo":"test"}]]`)

	for k,v := range target{
		if v != txdata[k]{
			log.Warn(" tx err : %d  %x   %x" ,k,v,txdata[k])
			//return
		}
	}
	txdata = txdata[:len(txdata)-1]
	if err != nil {
		fmt.Errorf(" MarshalBinary err :" + err.Error())
	}
	chainId, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")

	result := make([]byte,83)
	for i,v := range chainId{
		result[i] = v
	}
	for i,v := range txdata{
		result[i+32] = v
	}
	//result := eos.SigDigest(chainId, txdata, nil)
	log.Warn(result)
}
