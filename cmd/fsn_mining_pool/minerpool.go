package main

import (
	"fmt"
	"context"
	"crypto/ecdsa"
	"math/big"
	"sync"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/crypto"
	"github.com/FusionFoundation/efsn/log"
)

type MiningPool struct {
	Address common.Address
	Priv *ecdsa.PrivateKey
	Profit *big.Int
}

var mp *MiningPool

var mpLock *sync.Mutex = new(sync.Mutex)

func GetMiningPool() *MiningPool {
	if mp == nil {
		mpLock.Lock()
		defer mpLock.Unlock()
		if mp == nil {
			mp = &MiningPool{}
		}
	}
	return mp
}

func SetMiningPool(key *ecdsa.PrivateKey) {
	mp := GetMiningPool()
	mpLock.Lock()
	defer mpLock.Unlock()
	mp.Priv = key
	mp.Address = crypto.PubkeyToAddress(key.PublicKey)
}

func (mp *MiningPool) CalcProfit(after, before uint64) {
	log.Debug(fmt.Sprintf("mining pool calculating profit between %v and %v", after, before))
	mpLock.Lock()
	defer mpLock.Unlock()
	reward := GetBlocksReward(after, before, mp.Address)
	if reward == nil {
		mp.Profit = big.NewInt(0)
		return
	}
	mp.Profit = reward
}

func (mp *MiningPool) SendAsset(acc common.Address, asset *Asset) ([]common.Hash, error) {
	log.Debug("mining pool, SendAsset()", "to", acc, "asset", asset)
	mpLock.Lock()
	defer mpLock.Unlock()

	return sendAsset(mp.Address, acc, asset, mp.Priv)
}

func getBalance() *big.Int {
	client := GetRPCClient()
	bal, err := client.BalanceAt(context.Background(), mp.Address, nil)
	if err != nil {
		log.Warn("mining pool getBalance() failed", "error", err)
		return big.NewInt(0)
	}
	return bal
}
