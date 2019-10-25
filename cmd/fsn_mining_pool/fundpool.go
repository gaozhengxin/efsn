package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"time"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/crypto"
	"github.com/FusionFoundation/efsn/log"
)

type FundPool struct {
	Address common.Address
	Priv *ecdsa.PrivateKey
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

func SetFundPool(key *ecdsa.PrivateKey) {
	fp := GetFundPool()
	fpLock.Lock()
	defer fpLock.Unlock()
	fp.Priv = key
	fp.Address = crypto.PubkeyToAddress(key.PublicKey)
}

func (fp *FundPool) GetTotalOut(after, before uint64) (total *Asset, err error) {
	log.Debug("fund pool GetTotalOut")
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("GetTotalOut() failed, error: %v", r)
		}
	}()
	txs := GetTxFromAddress(fp.Address, after, before)
	if len(txs) > 0 {
		for _, tx := range txs {
			if tx.Receipt["fsnLogTopic"] == "SendAssetFunc" && tx.Receipt["AssetID"] == "0xffffffffffffffffffffffffffffffffffffffff" {
				amt := (*big.Int)(tx.Tx.Value)
				ast, _ := NewAsset(amt, uint64(time.Now().Unix()), 0)
				*total = *total.Add(ast)
			}
			if tx.Receipt["fsnLogTopic"] == "TimeLockFunc" && tx.Receipt["AssetID"] == "0xffffffffffffffffffffffffffffffffffffffff" {
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
	var hs []common.Hash
	var detained []Profit
	for _, p := range profits {
		log.Debug("fund pool send profit", "profit", p)
		ast, _ := NewAsset(p.Amount, 0, 0)
		hash, err := fp.SendAsset(p.Address, ast)
		if err != nil || hash == nil || len(hash) == 0 {
			detained = append(detained, p)
			continue
		}
		hs = append(hs, hash...)
	}
	return hs, detained
}

func (fp *FundPool) SendAsset(acc common.Address, asset *Asset) ([]common.Hash, error) {
	log.Debug("fund pool, SendAsset()", "to", acc, "asset", asset)
	fpLock.Lock()
	defer fpLock.Unlock()

	return sendAsset(fp.Address, acc, asset, fp.Priv)
}
