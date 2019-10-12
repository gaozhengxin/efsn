package main

import (
	"fmt"
	"math/big"
	"time"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/internal/ethapi"
	"github.com/FusionFoundation/efsn/log"
)

func main() {
	// SetMinerPoolAccount
	// SetFundPoolAccount
	mp := GetMinerPool()
	mp.Address = common.HexToAddress("0xd65f00dfd58d814f9de157fdddf6669677299c84")
	fp := GetFundPool()
	fp.Address = common.HexToAddress("0xd65f00dfd58d814f9de157fdddf6669677299c84")

	InitMongo()
	Run()
}

var (
	InitialBlock uint64 = 200000
	FeeRate = big.NewRat(1,10)  // 0.1
)

type UserAssetMap map[common.Address]*Asset

type Profit struct {
	Address common.Address
	Amount *big.Int
	Time int64
}

func DoDeposit(tx ethapi.TxAndReceipt) error {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("DoDeposit failed", "error", r)
		}
	}()

	from := tx.Tx.From
	start := tx.Receipt["fsnLogData"].(map[string]interface{})["StartTime"].(uint64)
	end := tx.Receipt["fsnLogData"].(map[string]interface{})["EndTime"].(uint64)
	amt := tx.Receipt["fsnLogData"].(map[string]interface{})["Value"].(*big.Int)
	asset, err := NewAsset(amt, start, end)
	if err != nil {
		log.Warn("DoDeposit failed", "error", err)
		return err
	}

	ast := GetUserAsset(from)
	if ast != nil {
		ast = (*ast).Add(asset)
	} else {
		ast = asset
	}
	err = SetUserAsset(from, *ast)
	if err != nil {
		log.Warn("DoDeposit failed", "error", err)
		return err
	}
	return nil
}

func DoWithdraw(m WithdrawMsg) error {

	// TODO chech m.TxHash

	ast := GetUserAsset(m.Address)
	if ast != nil {
		(*ast).Add(m.Asset)
	} else {
		ast = m.Asset
	}
	if !ast.IsNonneg() {
		log.Warn("DoWithdraw fail, account: %v has no enough asset", m.Address)
		return nil
	}
	err := SetUserAsset(m.Address, *ast)
	if err != nil {
		log.Warn("DoWithdraw failed", "error", err)
		return err
	}
	return nil
}

var WithdrawCh chan(WithdrawMsg) = make(chan WithdrawMsg)

type WithdrawMsg struct {
	TxHash common.Hash
	Address common.Address
	Asset *Asset
}

func NewWithdrawMsg(addr common.Address, amount *big.Int, start uint64, end uint64) WithdrawMsg {
	asset, _ := NewAsset(new(big.Int).Sub(big.NewInt(0), amount), start, end)
	return WithdrawMsg{
		Address: addr,
		Asset: asset,
	}
}

func SettleAccounts() error {
	mpLock.Lock()
	defer mpLock.Unlock()
	fpLock.Lock()
	defer fpLock.Unlock()

	mp := GetMinerPool()
	mp.CalcProfit()
	fp := GetFundPool()
	// 1. get fp out, replenish fp
	totalout, err := fp.GetTotalOut()
	if err != nil {
		return err
	}
	if totalout != nil {
		for i := 0; i < 3; i++ {
			log.Debug("Replenish fund pool", "try time", i, "amount", totalout)
			_, err := mp.SendFSN(fp.Address, totalout)
			if err == nil {
				break
			} else {
				log.Debug("Replenish fund pool failed", "try time", i, "error", err)
			}
		}
	}

	// 2. update lastSettlePoint
	h := GetHead()
	SetLastSettlePoint(h)

	// 3. calc userProfits
	// if withdraw, then no profit of the withdrawn part will be given in this day.
	totalProfit := mp.Profit
	uam := GetAllAssets()
	userProfits := CalculateUserProfits(totalProfit, uam)

	_, detained := fp.PayProfits(userProfits)
	for _, dp := range detained {
		err := AddDetainedProfit(dp)
		if err != nil {
			log.Warn("Write detained profit failed", "error", err)
			continue
		}
	}
	return nil
}

func CalculateUserProfits(totalProfit *big.Int, uam *UserAssetMap) []Profit {
	var userProfits []Profit
	day := time.Now().Round(time.Hour * 24).Add(-1 * time.Hour * 24).Unix()
	dayUserAsset := make(map[common.Address]*big.Int)
	for addr, a := range *uam {
		amt := a.GetAmountByTime(uint64(day))
		dayUserAsset[addr] = amt
		profit := Profit{
			Address: addr,
			Time: day,
		}
		userProfits = append(userProfits, profit)
	}
	totalAsset := big.NewInt(0)
	for _, a := range dayUserAsset {
		totalAsset = new(big.Int).Add(totalAsset, a)
	}
	for _, p := range userProfits {
		userAsset := dayUserAsset[p.Address]
		// totalProfit(userAsset/totalAsset)*(1 - FeeRate)
		u := new(big.Rat).SetInt(userAsset)
		t := new(big.Rat).SetInt(totalAsset)
		s := new(big.Rat).SetInt(totalProfit)
		up := new(big.Rat).Quo(u, t)
		up = new(big.Rat).Mul(up, s)
		up = new(big.Rat).Mul(up, new(big.Rat).Sub(big.NewRat(1,1), FeeRate))
		f, _ := up.Float64()
		p.Amount = big.NewInt(int64(f))
	}
	return userProfits
}

func GetTxType(tx ethapi.TxAndReceipt) (txtype string) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("GetTxType failed", "error", r)
			txtype = ""
		}
	}()
	mp := GetMinerPool()
	if tx.Receipt["fsnLogTopic"] == "TimeLockFunc" {
		if tx.Receipt["status"].(int) != 1 {
			return ""
		}
		to := tx.Receipt["fsnLogData"].(map[string]interface{})["To"].(common.Address)
		if to == mp.Address {
			return "DEPOSIT"
		}
	}
	return
}

func Run() {
	log.Info("mining pool running")
	examinetxstimer := time.NewTimer(5 * time.Second)
	timer := NewZeroTimer()

	h := GetHead()
	if h == 0 {
		SetHead(InitialBlock)
	}

	for {
		select {
		case <-examinetxstimer.C:
			// do every minute
			log.Info("examine transactions")
			ch := make(chan string)
			go func(ch chan string) {
				after := GetHead()
				before := GetSyncHead() - 10
				for before < InitialBlock || before <= after {
					log.Debug("cannot find new transactions in mongodb")
					time.Sleep(time.Second * 5)
				}
				// get txs from mongodb
				txs := GetTxs(after, before)
				for _, tx := range txs {
					txtype := GetTxType(tx)
					switch txtype {
					case "DEPOSIT":
						err := DoDeposit(tx)
						if err != nil {
							ch <- err.Error()
							return
						}
					default:
					}
				}
				var err error
				for i := 0; i < 3; i++ {
					err = SetHead(before - 10)
					if err == nil {
						break
					}
				}
				if err != nil {
					ch <- err.Error()
				} else {
					ch <- "ok"
				}
			}(ch)
			if ret := <-ch; ret == "ok" {
				log.Debug("renew timer")
				examinetxstimer = time.NewTimer(time.Minute)
			} else {
				panic(fmt.Errorf("examine txs error: %v", ret))
			}
		case m := <-WithdrawCh:
			// do withdraw
			log.Info("receive withdraw message")
			ch := make(chan string)
			go func(ch chan string) {
				err := DoWithdraw(m)
				if err != nil {
					ch <- err.Error()
				} else {
					ch <- "ok"
				}
			}(ch)
			if ret := <-ch; ret != "ok" {
				panic(fmt.Errorf("withdraw error: %v", ret))
			}
		case <-timer.C:
			// do everyday at 00:00
			log.Info("start settle accounts")
			ch := make(chan string)
			go func(chan string) {
				err := SettleAccounts()
				if err != nil {
					ch <- err.Error()
				} else {
					ch <- "ok"
				}
			}(ch)
			if ret := <-ch; ret == "ok" {
				timer = NewZeroTimer()
			} else {
				panic(fmt.Errorf("settle accounts error: %v", ret))
			}
		}
	}
}

func NewZeroTimer() *time.Timer {
	now := time.Now().Round(time.Hour * 24)
	next := now.Add(time.Hour * 24)
	timer := time.NewTimer(next.Sub(now))
	return timer
}
