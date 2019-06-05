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

import "testing"

func TestGetBlockNumber(t *testing.T) {

	tw := Client{
		BaseURL: "http://47.112.132.142:10004",
		Debug:   true,
	}

	if r, err := tw.getDynamicGlobal(); err != nil {
		t.Errorf("GetAccountNet failed: %v\n", err)
	} else {
		t.Logf("GetAccountNet return: \n\t%+v\n", r)
	}
	if r, err := tw.getGetBlock(1329514); err != nil {
		t.Errorf("GetAccountNet failed: %v\n", err)
	} else {
		t.Logf("GetAccountNet return: \n\t%+v\n", r)
	}

}

func TestGetBalance(t *testing.T) {

	tw := Client{
		BaseURL: "http://47.112.132.142:10004",
		Debug:   true,
	}

	if r, err := tw.GetBalance("kencani"); err != nil {
		t.Errorf("GetAccountNet failed: %v\n", err)
	} else {
		t.Logf("GetAccountNet return: \n\t%+v\n", r)
	}


}
