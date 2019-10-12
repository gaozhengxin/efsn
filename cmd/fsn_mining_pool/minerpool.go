package main

import (
	"math/big"
	"sync"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/log"
)

type MinerPool struct {
	Address common.Address
	// key manager
	Profit *big.Int
}

var mp *MinerPool

var mpLock *sync.Mutex = new(sync.Mutex)

func GetMinerPool() *MinerPool {
	if mp == nil {
		mpLock.Lock()
		defer mpLock.Unlock()
		if mp == nil {
			mp = &MinerPool{}
		}
	}
	return mp
}

// TODO
/*
func SetMinerPoolAccount(key manager) {
	fpLock.Lock()
	defer fpLock.Unlock()
}
*/

func (mp *MinerPool) CalcProfit() {
	mpLock.Lock()
	defer mpLock.Unlock()
	bal := GetMiningPoolBalance()
	newBal := getBalance()
	mp.Profit = new(big.Int).Sub(newBal, bal)
	SetMiningPoolBalance(newBal)
}

func (mp *MinerPool) SendFSN(acc common.Address, asset *Asset) (common.Hash, error) {
	log.Debug("miner pool, SendFSN()", "to", acc, "asset", asset)
	mpLock.Lock()
	defer mpLock.Unlock()
	// TODO
	mp.Balance = getBalance()
	return common.Hash{}, nil
}

func getBalance() *big.Int {
	// TODO
	return big.NewInt(0)
}
