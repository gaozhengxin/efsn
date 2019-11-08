package main

import (
	"context"
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
	Nonce uint64
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
	client := GetRPCClient()
	nonce, err := client.PendingNonceAt(context.Background(), fp.Address)
	if err != nil {
		log.Warn("get fund pool nonce error")
		fp.Nonce = uint64(1)
	} else {
		fp.Nonce = nonce
	}
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

func (fp *FundPool) PayProfits(profits []Profit) (map[string]mgoProfit, []Profit) {
	log.Debug("fund pool PayProfits()", "profits", profits)
	var m = make(map[string]mgoProfit)
	var detained []Profit
	for _, p := range profits {
		m[p.Address.Hex()] = mgoProfit{Amount:p.Amount.String()}
		log.Debug("fund pool send profit", "profit", p)
		if p.Amount.Cmp(big.NewInt(0)) == 0 {
			continue
		}
		ast, _ := NewAsset(p.Amount, 0, 0)
		hash := fp.SendAsset(p.Address, ast)
		if hash == nil || len(hash) == 0 {
			m[p.Address.Hex()]=mgoProfit{
				Amount:p.Amount.String(),
				Status:"failed",
			}
			detained = append(detained, p)
			continue
		}
		m[p.Address.Hex()]=mgoProfit{
			Hash:hash[0].Hex(),
			Amount:p.Amount.String(),
			Status:"success",
		}
		time.Sleep(time.Second * 5)
	}
	return m, detained
}

func (fp *FundPool) SendAsset(acc common.Address, asset *Asset) ([]common.Hash) {
	log.Debug("fund pool, SendAsset()", "to", acc, "asset", asset)
	fpLock.Lock()
	defer fpLock.Unlock()

	client := GetRPCClient()
	nonce, _ := client.PendingNonceAt(context.Background(), fp.Address)
	if nonce > fp.Nonce {
		fp.Nonce = nonce
	}
	hs, err := sendAsset(fp.Address, acc, asset, fp.Priv, &fp.Nonce)
	if err != nil {
		err = fmt.Errorf(err.Error() + "  " + fmt.Sprintf("from:%v, to:%v, asset:%+v", fp.Address.Hex(), acc.Hex(), *asset))
		AddError(err)
	}
	return hs
}
