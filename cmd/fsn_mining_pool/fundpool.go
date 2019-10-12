package main

import (
	"fmt"
	"math/big"
	"sync"
	"time"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/log"
)

type FundPool struct {
	Address common.Address
	// key manager
}

var  fp *FundPool

var fpLock *sync.Mutex = new(sync.Mutex)

func GetFundPool() *FundPool {
	if fp == nil {
		fpLock.Lock()
		defer fpLock.Unlock()
		if fp == nil {
			fp = &FundPool{}
		}
	}
	return fp
}

// TODO
/*
func SetFundPoolAccount(key manager) {
	fpLock.Lock()
	defer fpLock.Unlock()
}
*/

func (fp *FundPool) GetTotalOut() (total *Asset, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("GetTotalOut() failed, error: %v", r)
		}
	}()
	after := GetLastSettlePoint()
	txs := GetTxFromAddress(fp.Address, after)
	if len(txs) > 0 {
		for _, tx := range txs {
			if tx.Receipt["fsnLogTopic"] == "SendAssetFunc" {
				amt := (*big.Int)(tx.Tx.Value)
				ast, _ := NewAsset(amt, uint64(time.Now().Unix()), 0)
				*total = *total.Add(ast)
			}
			if tx.Receipt["fsnLogTopic"] == "TimeLockFunc" {
				start := tx.Receipt["fsnLogData"].(map[string]interface{})["StartTime"].(uint64)
				end := tx.Receipt["fsnLogData"].(map[string]interface{})["EndTime"].(uint64)
				amt := tx.Receipt["fsnLogData"].(map[string]interface{})["Value"].(*big.Int)
				ast, _ := NewAsset(amt, start, end)
				*total = *total.Add(ast)
			}
		}
	}
	return
}

func (fp *FundPool) PayProfits(profits []Profit) ([]common.Hash, []Profit) {
	log.Debug("fund pool PayProfits()", "profits", profits)
	fpLock.Lock()
	defer fpLock.Unlock()
	var hs []common.Hash
	var detained []Profit
	for _, p := range profits {
		log.Debug("fund pool send profit", "profit", p)
		ast, _ := NewAsset(p.Amount, 0, 0)
		hash, err := fp.SendFSN(p.Address, ast)
		if err != nil {
			detained = append(detained, p)
			continue
		}
		hs = append(hs, hash)
	}
	return hs, detained
}

func (fp *FundPool) SendFSN(acc common.Address, asset *Asset) (common.Hash, error) {
	log.Debug("fund pool, SendFSN()", "to", acc, "asset", asset)
	fpLock.Lock()
	defer fpLock.Unlock()
	// TODO
	return common.Hash{}, nil
}
