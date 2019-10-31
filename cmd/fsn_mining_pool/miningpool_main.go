package main

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"sync"
	"time"
	"github.com/FusionFoundation/efsn/accounts/keystore"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/internal/ethapi"
	"github.com/FusionFoundation/efsn/log"
	"github.com/spf13/cobra"
)

func initCmd() {
	rootCmd.PersistentFlags().StringVar(&node, "node", "http://0.0.0.0:8554", "efsn node rpc address")
	rootCmd.PersistentFlags().StringVar(&MongoIP, "mongo", "localhost", "mongoDB address")
	rootCmd.PersistentFlags().StringVar(&port, "port", "9990", "withdraw port")
	rootCmd.PersistentFlags().Uint64Var(&InitialBlock, "initialblock", uint64(1), "initial block number")
	rootCmd.PersistentFlags().Int64Var(&FeePercentage, "feepercentage", int64(10), "fee percentage")
	rootCmd.PersistentFlags().StringVar(&mpkeyfile, "mkey", "", "mining pool keyfile path")
	rootCmd.PersistentFlags().StringVar(&mppassphrase, "mpasswd", "", "mining pool keyfile password")
	rootCmd.PersistentFlags().StringVar(&fpkeyfile, "fkey", "", "fund pool keyfile path")
	rootCmd.PersistentFlags().StringVar(&fppassphrase, "fpasswd", "", "fund pool keyfile password")
	rootCmd.PersistentFlags().BoolVar(&mainnet, "main", true, "main net")
	rootCmd.PersistentFlags().BoolVar(&test, "test", false, "test net")
	rootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "private chain")
	rootCmd.PersistentFlags().IntVar(&verbose, "verbose", 3, "log verbose")
}

func initApp() {
	initCmd()
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
}

var (
	mpkeyfile string
	mppassphrase string
	fpkeyfile string
	fppassphrase string

	node string
	InitialBlock uint64
	FeePercentage int64
	FeeRate *big.Rat

	ChainID int64
	mainnet,test,dev bool

	verbose int
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
	if verbose > 5 {
		verbose = 5
	}
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbose), log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	if FeePercentage > 100 {
		log.Error("fee percentage cannot be larger than 100")
	}
	FeeRate = big.NewRat(FeePercentage, 100)

	c := GetRPCClient()
	if c == nil {
		log.Warn("cannot conncet to efsn node")
	}

	InitMongo()
	if Session == nil {
		return
	}

	if dev {
		ChainID = int64(55555)
	} else if test {
		ChainID = int64(46688)
	} else if mainnet {
		ChainID = int64(32659)
	} else {
		log.Error("unknown evironment")
	}

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
	log.Info("app is prepared", "mining pool", mp.Address.Hex(), "fund pool", fp.Address.Hex(), "fee rate", FeeRate.FloatString(2), "start at block number", InitialBlock, "connecting efsn node", node, "listening withdraw message", port, "chain id", ChainID, "log verbose", log.Lvl(verbose))

	cb := GetCoinbase()
	if cb != mp.Address {
		log.Warn("node coinbase is not set to mining pool address", "coinbase", cb.Hex())
	}

	bal1 := GetBalance(mp.Address)
	bal2 := GetBalance(fp.Address)
	fmt.Printf("bal1:%+v, bal2:%+v\n", bal1, bal2)
	tbal1 := GetTimelockBalance(mp.Address)
	tbal2 := GetTimelockBalance(fp.Address)
	fmt.Printf("tbal1:%+v, tbal2:%+v\n", tbal1, tbal2)

	if IsMining() == false {
		log.Info("node is not mining")
		if IsAutoBuyTicket() == true {
			log.Warn("\x1b[41m detected node is auto buying ticket buy is not mining!!! try stop auto buying ticket \x1b[0m")
			ok := StopAutoBuyTicket()
			if ok == false {
				log.Warn("\x1b[41m stop auto buying ticket failed!!! \x1b[0m")
			}
		}
	} else {
		log.Info("node is mining")
		if IsAutoBuyTicket() == false {
			log.Info("node is auto buying ticket")
		} else {
			log.Info("node is not auto buying ticket")
		}
	}

	go ServerRun()
	go Run()
	select{}
}

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
	if t >= 9223372036854775807 {
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
	start := uint64(0)
	end := uint64(0)
	mp := GetMiningPool()
	if tx.Receipt["fsnLogTopic"] == "TimeLockFunc" {
		start = ParseTime(tx.Receipt["fsnLogData"].(map[string]interface{})["StartTime"])
		end = ParseTime(tx.Receipt["fsnLogData"].(map[string]interface{})["EndTime"])
	}
	amt := new(big.Int)
	if tx.Receipt["fsnLogData"].(map[string]interface{})["Value"] != nil {
		amt = tx.Receipt["fsnLogData"].(map[string]interface{})["Value"].(*big.Int)
	} else if value := tx.Tx.Value; *tx.Tx.To == mp.Address && value != nil {
		amt = value.ToInt()
	}
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
	AddDeposit(tx.Receipt["transactionHash"].(common.Hash), from, ast)
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
	bal := GetBalance(fp.Address)
	if bal.Sub(m.Asset).IsNonneg() == false {
		// 资金池不够, 停止买票, 等待矿池timelock余额够了再发给用户
		log.Info("user withdraw request is accepted but fund pool has no enough balance to withdraw")
		ret := &WithdrawRet{
			Id: m.Id,
			Msg: "withdraw request is accepted, refund will take place in minutes",
		}
		WithdrawRetCh <- *ret

		WithdrawLock.Lock()

		StopAutoBuyTicket()
		timer := time.NewTimer(time.Minute * 30)
		timeout := false
		go func() {
			<-timer.C
			timeout = true
		}()
		for {
			if timeout == true {
				log.Warn("mining pool pause timeout, do withdraw failed")
				break
			}
			mpbal := GetTimelockBalance(mp.Address)
			if mpbal != nil {
				if mpbal.Sub(m.Asset).IsNonneg() {
					hs0, _ := mp.SendAsset(fp.Address, m.Asset)
					AddMiningPoolToFundPool(hs0, m.Asset)
					hs, err := fp.SendAsset(m.Address, m.Asset) // timelock to timelock
					if err != nil || hs == nil || len(hs) == 0 {
						log.Warn("DoWithdraw send asset failed", "error", err)
						WithdrawLock.Unlock()
						return
					}
					p := GetLastSettlePoint()
					err = AddWithdraw(hs[0], m, p)
					if err != nil {
						log.Warn("DoWithdraw success but write record failed", "error", err)
					}
					break
				}
			}
			time.Sleep(time.Second * 1)
		}
		if MustStartMining() == true {
			if StartAutoBuyTicket() == false {
				log.Warn("node is mining but start auto buy ticket failed")
			} else {
				log.Info("node is mining and auto buying ticket")
			}
		} else {
			log.Warn("node is not mining and start mining failed")
		}
		WithdrawLock.Unlock()
	} else {
		// 资金池有足够asset, 直接发给用户
		hs, err := fp.SendAsset(m.Address, m.Asset)

		ret := &WithdrawRet{
			Hs: hs,
			Id: m.Id,
		}
		WithdrawRetCh <- *ret
		if err != nil || hs == nil || len(hs) == 0 {
			log.Warn("DoWithdraw send asset failed", "error", err)
			return
		}
		p := GetLastSettlePoint()
		err = AddWithdraw(hs[0], m, p)
		if err != nil {
			log.Warn("DoWithdraw success but write record failed", "error", err)
		}
	}
	return
}

var WithdrawCh chan(WithdrawMsg) = make(chan WithdrawMsg)
var WithdrawLock = new(sync.Mutex)
var WithdrawRetCh = make(chan WithdrawRet)

type WithdrawMsg struct {
	Address common.Address
	Asset *Asset
	Id int
}

type WithdrawRet struct {
	Hs []common.Hash `json:"hashes,omitempty"`
	Id int `json:"id"`
	Msg string `json:"msg,omitempty"`
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
	WithdrawLock.Lock()
	defer WithdrawLock.Unlock()

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
	totalProfit := mp.Profit
	if err := AddTotalProfit(p0, p1, totalProfit); err != nil {
		log.Warn("write total profit to mongo failed", "error", err)
	}

	// 2. send profit to fund pool
	ast, err := NewAsset(totalProfit, 0, 0)
	if ast != nil && err == nil {
		log.Info(fmt.Sprintf("mining pool profit is %v", ast))
		hs, err := mp.SendAsset(fp.Address, ast)
		AddMiningPoolToFundPool(hs, ast)
		if err != nil {
			log.Warn("Replenish fund pool reported error", "error", err)
		}
	} else {
		log.Warn("cannot convert profit to asset", "error", err)
	}

	// 3. calc and pay userProfits
	// if withdraw, then no profit of the withdrawn part will be given in this day.
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
	AddProfit(p0, p1, hs)
	log.Debug(fmt.Sprintf("fund pool has commited %v transactions", len(hs)), "hashes", fmt.Sprintf("%+v", hs))
	for _, dp := range detained {
		err := AddDetainedProfit(dp)
		if err != nil {
			log.Warn("Write detained profit failed", "error", err)
			continue
		}
	}

	// 4. get refund data, replenish fp
	ws := GetWithdrawByPhase(p0)
	refund := ZeroAsset()
	for _, ws := range ws {
		if ws.Asset != nil {
			refund = refund.Add(ws.Asset)
		}
	}
	if refund.Equal(ZeroAsset()) == false {
		log.Debug("replenish fund pool")
		StopAutoBuyTicket()
		timer := time.NewTimer(time.Minute * 30)
		timeout := false
		go func() {
			<-timer.C
			timeout = true
		}()
		for {
			if timeout == true {
				log.Warn("mining pool pause timeout, replenish fund pool failed")
				break
			}
			mpbal := GetTimelockBalance(mp.Address)
			if mpbal != nil {
				if mpbal.Sub(refund).IsNonneg() == true {
					hs, err := mp.SendAsset(fp.Address, refund)
					AddMiningPoolToFundPool(hs, refund)
					if err != nil {
						log.Warn("send timelock to fund pool failed", "error", err)
						return fmt.Errorf("replenish fund pool failed, %v", err)
					}
					hsh := ""
					for _, s := range hs {
						hsh = hsh + ", " + s.Hex()
					}
					log.Info("replenish fund pool finished", "hashes", hsh)
					return nil
				}
			}
		}
	}
	return nil
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
	if tx.Receipt["fsnLogTopic"] == "TimeLockFunc" || tx.Receipt["fsnLogTopic"] == "SendAssetFunc" {
		if tx.Receipt["status"].(int) != 1 {
			return ""
		}
		to := tx.Receipt["fsnLogData"].(map[string]interface{})["To"].(common.Address)
		if to == mp.Address {
			return "DEPOSIT"
		}
	}
	if tx.Receipt["fsnLogTopic"] == "" && *tx.Tx.To == mp.Address {
		return "DEPOSIT"
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
					before = GetSyncHead()
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
