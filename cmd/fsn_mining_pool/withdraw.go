package main

import (
	"fmt"
	"math/big"
	"strconv"
	"time"
	"github.com/FusionFoundation/efsn/cmd/fsn_mining_pool/withdraw"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/log"
)

func ValidateWithdraw(r *withdraw.WithdrawRequest) error {
	log.Info("start ValidateWithdraw()", "request", r)
	if common.HexToHash(r.Hash) != r.MakeHash() {
		return fmt.Errorf("withdraw request hash error")
	}
	if ts, err := strconv.ParseUint(r.Timestamp, 10, 64); err == nil {
		now := time.Now().Unix()
		if uint64(now) - ts > 300 {
			return fmt.Errorf("timestamp is too old")
		}
		if ts > 60 + uint64(now) {
			return fmt.Errorf("timestamp is too ahead of time")
		}
	} else {
		return fmt.Errorf("invalid timestamp")
	}
	if r.VerifySignature() {
		log.Debug("signature verify passed")
		user := common.HexToAddress(r.Address)
		amount, ok := new(big.Int).SetString(r.Amount, 10)
		if !ok || amount.Cmp(big.NewInt(0)) <1 {
			return fmt.Errorf("withdraw amount error")
		}
		// 获取余额
		userasset := GetUserAsset(user)
		if userasset == nil {
			return fmt.Errorf("user not found")
		}
		// 选最长的一段
		today := GetTodayZero().Unix()
		sendasset, _ := NewAsset(amount, uint64(today), 0)
		rem := userasset.Sub(sendasset)
		log.Debug("ValidateWithdraw()", "rem", rem)
		starttime := (*rem)[0].T
		endtime := starttime
		for i := 0; i < len(*rem); i++ {
			endtime = (*rem)[i].T
			if (*rem)[i].V.Cmp(big.NewInt(0)) >= 0 {
				continue
			} else {
				break
			}
		}
		log.Debug("ValidateWithdraw()", "starttime", starttime, "endtime", endtime)
		if starttime == endtime {
			return fmt.Errorf("no enough balance")
		}
		// 构造Asset
		sendasset, _ = NewAsset(amount, starttime, endtime)
		log.Debug("ValidateWithdraw() asset prepared", "asset", sendasset)
		// 构造WithdrawMsg
		msg := WithdrawMsg{
			Address:user,
			Asset:sendasset,
		}
		// AddWithdrawLog
		AddWithdrawLog(*r)
		// 传入WithdrawCh
		WithdrawCh <- msg
	} else {
		return fmt.Errorf("verify signature failed")
	}
	return nil
}
