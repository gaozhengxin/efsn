package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"
	"github.com/FusionFoundation/efsn/accounts/keystore"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/internal/ethapi"
	"github.com/FusionFoundation/efsn/log"
	"github.com/spf13/cobra"
)

func initCmd() {
	rootCmd.PersistentFlags().StringVar(&mpkeyfile, "mkey", "", "mining pool keyfile path")
	rootCmd.PersistentFlags().StringVar(&mppassphrase, "mpasswd", "", "mining pool keyfile password")
	rootCmd.PersistentFlags().StringVar(&fpkeyfile, "fkey", "", "fund pool keyfile path")
	rootCmd.PersistentFlags().StringVar(&fppassphrase, "fpasswd", "", "fund pool keyfile password")
}

func initApp() {
	initCmd()
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	InitMongo()
}

var (
	mpkeyfile string
	mppassphrase string
	fpkeyfile string
	fppassphrase string
)

var rootCmd = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		runApp()
	},
}

func main() {
	initApp()
	if err := rootCmd.Execute(); err != nil {
		log.Error(err.Error())
	}
}

func runApp() {
	keyjson1, err1 := ioutil.ReadFile(mpkeyfile)
	if err1 != nil {
		log.Error("read mining pool key file failed", "error", err1)
	}
	k1, err1 := keystore.DecryptKey(keyjson1, mppassphrase)
	if err1 != nil {
		log.Error("decrypt mining pool key failed", "error", err1)
	}

	keyjson2, err2 := ioutil.ReadFile(fpkeyfile)
	if err2 != nil {
		log.Error("read fund pool key file failed", "error", err1)
	}
	k2, err2 := keystore.DecryptKey(keyjson2, fppassphrase)
	if err2 != nil {
		log.Error("decrypt fund pool key failed", "error", err2)
	}

	SetMiningPool(k1.PrivateKey)
	SetFundPool(k2.PrivateKey)
	mp := GetMiningPool()
	fp := GetFundPool()
	log.Info("app is prepared", "mining pool", mp.Address.Hex(), "fund pool", fp.Address.Hex())

	go ServerRun()
	go Run()
	select{}
}

var (
	url string = "http://0.0.0.0:8554"
	InitialBlock uint64 = 10
	//InitialBlock uint64 = 200000
	FeeRate = big.NewRat(1,10)  // 0.1
)

type UserAssetMap map[common.Address]*Asset

type Profit struct {
	Address common.Address
	Amount *big.Int
	Time int64
}

func ParseTime(input interface{}) (t uint64) {
	defer func() {
		if r := recover(); r != nil {
			t = 0
		}
	}()
	t = input.(uint64)
	if t >= 9223372036854775808 {
		return 0
	}
	return
}

func DoDeposit(tx ethapi.TxAndReceipt) error {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("DoDeposit failed", "error", r)
		}
	}()

	from := tx.Tx.From
	start := ParseTime(tx.Receipt["fsnLogData"].(map[string]interface{})["StartTime"])
	end := ParseTime(tx.Receipt["fsnLogData"].(map[string]interface{})["EndTime"])
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

func DoWithdraw(m WithdrawMsg) {
	ast := GetUserAsset(m.Address)
	if ast != nil {
		ast = ast.Sub(m.Asset)
	} else {
		ast = m.Asset
	}
	ast.Reduce()
	today := GetTodayZero().Unix()
	ast.Align(uint64(today))
	if !ast.IsNonneg() {
		log.Warn("DoWithdraw fail, account: %v has no enough asset", m.Address)
		ret := &WithdrawRet{
			Id: m.Id,
			Error: fmt.Errorf("%v has no enough asset", m.Address),
		}
		WithdrawRetCh <- *ret
		return
	}
	err := SetUserAsset(m.Address, *ast)
	if err != nil {
		log.Warn("DoWithdraw failed", "error", err)
		ret := &WithdrawRet{
			Id: m.Id,
			Error: err,
		}
		WithdrawRetCh <- *ret
		return
	}
	log.Info("DoWithdraw send asset to user")
	fp := GetFundPool()
	hs, err := fp.SendAsset(m.Address, ast)
	if err != nil {
		log.Warn("DoWithdraw send asset failed", "error", err)
	}
	ret := &WithdrawRet{
		Hs: hs,
		Id: m.Id,
	}
	WithdrawRetCh <- *ret
	return
}

var WithdrawCh chan(WithdrawMsg) = make(chan WithdrawMsg)
var WithdrawRetCh = make(chan WithdrawRet)

type WithdrawMsg struct {
	Address common.Address
	Asset *Asset
	Id int
}

type WithdrawRet struct {
	Hs []common.Hash `json:"hashes,omitempty"`
	Id int `json:"id"`
	Error error `json:"error,omitempty"`
}

// SettleAccounts runs every day 0:00
// first, calculate mining profit in the last settlement peroid
// then calculates fund pool output among
// last settle point (block height, kept in mongo)
// and current head (block height)
// then updates last settle point for next peroid
// then calculates and pays every user's profit according to policy
func SettleAccounts() error {
	mp := GetMiningPool()
	fp := GetFundPool()

	// 0. decide settle interval
	p0 := GetLastSettlePoint()
	if p0 < InitialBlock {
		p0 = InitialBlock
	}
	p1 := GetHead()
	defer SetLastSettlePoint(p1)
	log.Info(fmt.Sprintf("do settlement between %v and %v", p0, p1))

	// 1. calc mining pool profit
	mp.CalcProfit(p0, p1)

	// 2. get fp out, replenish fp
	totalout, err := fp.GetTotalOut(p0, p1)
	if err != nil {
		log.Warn("SettleAccounts get fund pool out failed", "error", err)
	}
	if totalout != nil {
		log.Info(fmt.Sprintf("fund pool total out is %v", totalout))
		_, err := mp.SendAsset(fp.Address, totalout)
		if err != nil {
			log.Warn("Replenish fund pool reported error", "error", err)
		}
	} else {
		log.Info("no output from fund pool")
	}

	// 3. calc and pay userProfits
	// if withdraw, then no profit of the withdrawn part will be given in this day.
	totalProfit := mp.Profit
	log.Info(fmt.Sprintf("mining pool total profit is %v", totalProfit))
	if totalProfit.Cmp(big.NewInt(0)) <= 0 {
		log.Info("no total profit in mining pool in last settlement peroid")
		return nil
	}
	uam := GetAllAssets()
	if uam == nil {
		return fmt.Errorf("cannot get all users asset")
	}
	userProfits := CalculateUserProfits(totalProfit, uam)

	hs, detained := fp.PayProfits(userProfits)
	hashes := ""
	if len(hs) > 0 {
		for _, h := range hs {
			hashes = hashes + "  " + h.Hex()
		}
	}
	log.Debug(fmt.Sprintf("fund pool has commited %v transactions", len(hs)), "hashes", hashes)
	for _, dp := range detained {
		err := AddDetainedProfit(dp)
		if err != nil {
			log.Warn("Write detained profit failed", "error", err)
			continue
		}
	}

	return err
}

func CalculateUserProfits(totalProfit *big.Int, uam *UserAssetMap) []Profit {
	var userProfits []Profit
	day := time.Now().Add(-1 * time.Hour).Unix()
	dayUserAsset := make(map[common.Address]*big.Int)
	for addr, a := range *uam {
		amt := a.GetAmountByTime(uint64(day))
		dayUserAsset[addr] = amt
		profit := Profit{
			Address: addr,
			Time: day,
			Amount: big.NewInt(0),
		}
		userProfits = append(userProfits, profit)
	}
	totalAsset := big.NewInt(0)
	for _, a := range dayUserAsset {
		totalAsset = new(big.Int).Add(totalAsset, a)
	}
	if totalAsset.Cmp(big.NewInt(0)) < 1 {
		return userProfits
	}
	for i := 0; i < len(userProfits); i++ {
		p := userProfits[i]
		userAsset := dayUserAsset[p.Address]
		// totalProfit(userAsset/totalAsset)*(1 - FeeRate)
		u := new(big.Rat).SetInt(userAsset)
		t := new(big.Rat).SetInt(totalAsset)
		s := new(big.Rat).SetInt(totalProfit)
		up := new(big.Rat).Quo(u, t)
		up = new(big.Rat).Mul(up, s)
		up = new(big.Rat).Mul(up, new(big.Rat).Sub(big.NewRat(1,1), FeeRate))
		p.Amount = new(big.Int).Quo(up.Num(), up.Denom())
		userProfits[i] = p
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
	mp := GetMiningPool()
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
	examinetxstimer := time.NewTimer(time.Second * 5)
	timer := NewZeroTimer()

	h := GetHead()
	if h == 0 {
		log.Info("set head to initial block", "initial block", InitialBlock)
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
				before := GetSyncHead()
				log.Debug(fmt.Sprintf("try to fetch txs between %v and %v", after, before))
				for before < InitialBlock || before <= after {
					log.Debug("cannot find new transactions in mongodb")
					time.Sleep(time.Second * 5)
				}
				// get txs from mongodb
				txs := GetTxs(after, before)
				log.Debug(fmt.Sprintf("found %v relavant transactions", len(txs)))
				for _, tx := range txs {
					txtype := GetTxType(tx)
					fmt.Println("txtype:" + txtype)
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
				for i := 0; i < 30; i++ {
					err = SetHead(before)
					if err == nil {
						break
					}
					time.Sleep(time.Second * 5)
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
			fmt.Println("receive withdraw message")
			log.Info("receive withdraw message")
			DoWithdraw(m)
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
				log.Debug("settlement finished with no error")
				time.Sleep(time.Minute)
				timer = NewZeroTimer()
			} else {
				panic(fmt.Errorf("settle accounts error: %v", ret))
			}
		}
	}
}
