package main

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"github.com/FusionFoundation/efsn/cmd/fsn_mining_pool/withdraw"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/common/hexutil"
	"github.com/FusionFoundation/efsn/internal/ethapi"
	"github.com/FusionFoundation/efsn/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	Session *mgo.Session
	InitOnce bool
	database *mgo.Database
	MongoIP string = "localhost" // default port: 27017
	dbname string = "fusion"
)

var AssetsLock *sync.Mutex = new(sync.Mutex)

func InitMongo() {
	if InitOnce {
		return
	}
	if Session != nil {
		Session.Refresh()
	}
	InitOnce = true
	url := fmt.Sprintf("mongodb://%v", MongoIP) //url := "localhost"
	//url := "192.168.1.127:27017"
	fmt.Printf("mongodb url %v\n", url)
	for {
		session, err := mgo.Dial(url)
		if err != nil {
			log.Warn("mgo.Dial", "url", url, "fail", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		Session = session
		break
	}
	Session.SetMode(mgo.Monotonic, true)
	Session.SetSocketTimeout(1 * time.Hour)
	database = Session.DB(dbname)
	fmt.Printf("mongodb mongoServerInit finished.\n")
	InitOnce = false
}

func GetTxs(after, before uint64) []ethapi.TxAndReceipt {
	log.Debug("mongo GetTxs()", "after", after, "before", before)
	mp := GetMiningPool()
	address := mp.Address.Hex()
	collectionTable := database.C("Transactions")
	d := make([]ethapi.TxAndReceipt, 0)
	dd := make([]interface{}, 0)
	err := collectionTable.Find(bson.M{"receipt.fsnLogTopic":"TimeLockFunc", "receipt.fsnLogData.AssetID":"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "receipt.fsnLogData.To":bson.M{"$regex":address,"$options":"i"}, "tx.blockNumber":bson.M{"$gte":after,"$lt":before}}).All(&dd)
	if err != nil {
		if err.Error() != "not found" {
			log.Warn("mongo GetTxs() ", "error", err)
		}
		return nil
	}
	fmt.Printf("\nlen(dd) is %v\n", len(dd))
	if len(dd) > 0 {
		for _, obj := range dd {
			tx, err := ParseTxAndReceipt(obj.(bson.M))
			if err != nil {
				log.Warn("parse TxAndReceipt error", "obj", obj, "error", err)
				continue
			}
			d = append(d, tx)
		}
	}
	return d
}

func GetTxFromAddress(addr common.Address, after, before uint64) []ethapi.TxAndReceipt {
	log.Debug("mongo GetTxFromAddress", "address", addr, "after block", after)
	fp := GetFundPool()
	address := fp.Address.Hex()
	collectionTable := database.C("Transactions")
	d := make([]ethapi.TxAndReceipt, 0)
	dd := make([]interface{}, 0)
	err := collectionTable.Find(bson.M{"tx.From":bson.M{"$regex":address,"$options":"i"}, "tx.blockNumber":bson.M{"$gte":after,"$lt":before}}).All(&dd)
	if err != nil {
		log.Warn("mongo GetTxFromAddress() ", "error", err)
		return nil
	}
	if len(dd) > 0 {
		for _, obj := range dd {
			tx, err := ParseTxAndReceipt(obj.(bson.M))
			if err != nil {
				log.Warn("parse TxAndReceipt error", "obj", obj, "error", err)
				continue
			}
			d = append(d, tx)
		}
	}
	return d
}

/*
func GetSyncHead() (sh uint64) {
	log.Debug("mongo GetSyncHead()")
	defer func() {
		if r := recover(); r != nil {
			log.Debug("GetSyncHead() failed", "error", r)
			sh = 0
		}
	}()
	collectionTable := database.C("Transactions")
	// db.Transactions.find().sort({'tx.blockNumber':-1}).skip(0).limit(1)
	d := make([]bson.M, 1)
	err := collectionTable.Find(bson.M{}).Sort("-tx.blockNumber").Skip(0).Limit(1).One(&d[0])
	if err != nil {
		log.Debug("GetSyncHead() failed", "error", err)
		return 0
	}
	if d[0] != nil {
		 sh = uint64(d[0]["tx"].(bson.M)["blockNumber"].(int64))
	}
	return sh
}
*/

func GetSyncHead() (sh uint64) {
	log.Debug("mongo GetSyncHead()")
	defer func() {
		if r := recover(); r != nil {
			log.Debug("GetSyncHead() failed",  "error", r)
			sh = 0
		}
	}()
	collectionTable := database.C("Blocks")
	d := make([]bson.M, 1)
	//db.Blocks.find({},{'number':1}).sort({'number':-1}).skip(0).limit(1)
	err := collectionTable.Find(bson.M{}).Select(bson.M{"number":1}).Sort("-number").Skip(0).Limit(1).One(&d[0])
	if err != nil {
		log.Debug("GetSyncHead() failed", "error", err)
		return 0
	}
	if d[0] != nil {
		sh = uint64(d[0]["number"].(int64))
	}
	return sh
}

func GetHead() (h uint64) {
	log.Debug("mongo GetHead()")
	defer func() {
		if r := recover(); r != nil {
			log.Warn("mongo GetHead() fail", "error", r)
			h = 0
		}
	}()
	collectionTable := database.C("Miningpool")
	d := make([]interface{}, 1)
	err := collectionTable.Find(bson.M{"_id":"head"}).One(&d[0])
	if err != nil {
		if err.Error() != "not found" {
			log.Warn("mongo GetHead() failed", "error", err)
		}
		return 0
	}
	if len(d) > 0 && d[0] != nil {
		h = uint64(d[0].(bson.M)["value"].(int64))
		return
	}
	return 0
}

func SetHead(h uint64) error {
	log.Debug("mongo SetHead()", "head", h)
	collectionTable := database.C("Miningpool")
	d := bson.M{"_id":"head","value":h}
	_, err := collectionTable.Upsert(bson.M{"_id":"head"}, d)
	if err != nil {
		log.Warn("mongo SetHead failed", "error", err)
	}
	return err
}

func GetLastSettlePoint() (p uint64) {
	log.Debug("mongo GetLastSettlePoint()")
	defer func() {
		if r := recover(); r != nil {
			log.Warn("mongo GetLastSettlePoint() fail", "error", r)
			p = 0
		}
	}()
	collectionTable := database.C("Miningpool")
	d := make([]interface{}, 1)
	err := collectionTable.Find(bson.M{"_id":"settlepoint"}).One(&d[0])
	if err != nil {
		if err.Error() != "not found" {
			log.Warn("mongo GetLastSettlePoint() fail", "error", err)
		}
		return 0
	}
	if len(d) > 0 && d[0] != nil {
		p = uint64(d[0].(bson.M)["value"].(int64))
		return
	}
	return 0
}

func SetLastSettlePoint(p uint64) error {
	log.Debug("mongo SetLastSettlePoint()", "head", p)
	collectionTable := database.C("Miningpool")
	d := bson.M{"_id":"settlepoint","value":p}
	_, err := collectionTable.Upsert(bson.M{"_id":"settlepoint"}, d)
	if err != nil {
		log.Warn("mongo SetLastSettlePoint failed", "error", err)
	}
	return err
}

func GetBlocksReward(a, b uint64, miner common.Address) *big.Int {
	log.Debug("mongo GetBlocksReward()")
	defer func() {
		if r := recover(); r != nil {
			log.Warn("GetBlocksReward failed", "error", r)
		}
	}()
	collectionTable := database.C("Blocks")
	d := make([]bson.M,0)
	//db.Blocks.find({'number':{$gte:100, $lt:1000}, 'miner':{$regex:"0x07f35aba9555a532c0edc2bd6350c891b6f2c8d0",$options:"i"}})
	err := collectionTable.Find(bson.M{"number":bson.M{"$gte":a,"$lt":b}, "miner":bson.M{"$regex":miner.Hex(),"$options":"i"}}).Select(bson.M{"miner":1, "reward":1}).All(&d)
	if err != nil {
		panic(err)
	}
	reward := big.NewInt(0)
	log.Debug(fmt.Sprintf("mining pool has mined %v blocks between %v and %v", len(d), a, b))
	for _, b := range d {
		if b["reward"] != nil {
			r, ok := new(big.Int).SetString(b["reward"].(string), 10)
			if !ok {
				log.Warn("reward string error", "reward", b["reward"])
				continue
			}
			reward = new(big.Int).Add(reward, r)
		}
	}
	return reward
}

func GetAllAssets() (uam *UserAssetMap) {
	AssetsLock.Lock()
	defer AssetsLock.Unlock()
	defer func() {
		if r := recover(); r != nil {
			log.Warn("GetAllAssets error", "error", r)
		}
	}()
	uam = new(UserAssetMap)
	*uam = make(map[common.Address]*Asset)
	collectionTable := database.C("Assets")
	c, _ := collectionTable.Find(bson.M{}).Count()
	d := make([]bson.M, c)
	err := collectionTable.Find(bson.M{}).All(&d)
	if err != nil {
		if err.Error() != "not found" {
			log.Warn("mongo GetAllAssets() failed", "error", err)
		}
		return
	}
	for _, doc := range d {
		mgoast := mgoAsset{}
		for i := 0; i < len(doc["asset"].([]interface{})); i ++ {
			data, _ := bson.Marshal(doc["asset"].([]interface{})[i])
			p := &mgoPoint{}
			err = bson.Unmarshal(data, p)
			if err != nil {
				log.Warn("mongo GetUserAsset() fail", "error", err)
				break
			}
			mgoast = append(mgoast, *p)
		}
		ast := &Asset{}
		*ast = ConvertMGOAsset(mgoast)

		ast.Sort()
		ast.Reduce()

		(*uam)[common.HexToAddress(doc["_id"].(string))] = ast
	}
	return uam
}

func SetUserAsset(usr common.Address, ast Asset) error {
	AssetsLock.Lock()
	defer AssetsLock.Unlock()
	log.Debug("mongo SetUserAsset()", "user", usr, "asset", ast)

	ast.Sort()
	ast.Reduce()
	today := GetTodayZero().Unix()
	ast.Align(uint64(today))
	ast.Reduce()

	collectionTable := database.C("Assets")
	id := strings.ToLower(usr.Hex())
	mgoast := ConvertAsset(ast)
	d := bson.M{"_id":id, "asset":mgoast}
	_, err := collectionTable.Upsert(bson.M{"_id":id}, d)
	return err
}

func GetUserAsset(usr common.Address) *Asset {
	AssetsLock.Lock()
	defer AssetsLock.Unlock()
	log.Debug("mongo GetUserAsset()")
	collectionTable := database.C("Assets")
	id := strings.ToLower(usr.Hex())
	d := make([]bson.M, 1)
	err := collectionTable.Find(bson.M{"_id":id}).One(&d[0])
	if err != nil {
		if err.Error() != "not found" {
			log.Warn("mongo GetUserAsset() fail", "error", err)
		}
		return nil
	}
	mgoast := mgoAsset{}
	for i := 0; i < len(d[0]["asset"].([]interface{})); i ++ {
		data, _ := bson.Marshal(d[0]["asset"].([]interface{})[i])
		p := &mgoPoint{}
		err = bson.Unmarshal(data, p)
		if err != nil {
			log.Warn("mongo GetUserAsset() fail", "error", err)
			return nil
		}
		mgoast = append(mgoast, *p)
	}
	ast := &Asset{}
	*ast = ConvertMGOAsset(mgoast)

	ast.Sort()
	ast.Reduce()

	return ast
}

func AddWithdrawLog(req withdraw.WithdrawRequest) error {
	log.Debug("mongo AddWithdrawLog()")
	collectionTable := database.C("WithdrawLogs")
	d := bson.M{"withdrawrequest":req}
	err := collectionTable.Insert(d)
	return err
}

func AddWithdraw(h common.Hash, m WithdrawMsg) error {
	log.Debug("mongo AddWithdraw()")
	collectionTable := database.C("Withdraw")
	d := bson.M{"txhash":h.Hex(), "withdraw":m}
	err := collectionTable.Insert(d)
	return err
}

func AddDeposit(txhash common.Hash, user common.Address, ast *Asset) error {
	log.Debug("mongo AddWithdraw()")
	collectionTable := database.C("Deposit")
	mgoast := ConvertAsset(*ast)
	d := bson.M{"txhash":txhash.Hex(), "user":user, "asset":mgoast}
	err := collectionTable.Insert(d)
	return err
}

func AddDetainedProfit(p Profit) error {
	log.Debug("mongo AddDetainedProfit()", "profit", p)
	collectionTable := database.C("DetainedProfits")
	mgop := ParseProfit(p)
	err := collectionTable.Insert(mgop)
	return err
}

type MgoProfit struct {
	Address string `bson:"address"`
	Amount string `bson:"amount"`
	Time int64 `bson:"time"`
}

func ParseProfit(p Profit) MgoProfit {
	return MgoProfit{
		Address:p.Address.Hex(),
		Amount:p.Amount.String(),
		Time:p.Time,
	}
}

func SetMiningPoolBalance(bal *big.Int) {
	log.Debug("mongo SetMiningPoolBalance()", "bal", bal)
	collectionTable := database.C("Miningpool")
	collectionTable.Upsert(bson.M{"_id":"balance"}, bson.M{"_id":"balance","value":bal.String()})
}

func GetMiningPoolBalance() *big.Int {
	log.Debug("mongo GetMiningPoolBalance()")
	collectionTable := database.C("Miningpool")
	d := make([]bson.M, 1)
	err := collectionTable.Find(bson.M{"_id":"balance"}).One(&d[0])
	if err != nil {
		if err.Error() != "not found" {
			log.Warn("mongo GetMiningPoolBalance() failed", "error", err)
		}
		return nil
	}
	if d[0]["value"] != nil {
		bal, _ := new(big.Int).SetString(d[0]["value"].(string), 10)
		return bal
	}
	return nil
}

type mgoAsset []mgoPoint

type mgoPoint struct {
	T uint64 `bson:"T"`
	V string `bson:"V"`
}

func ConvertAsset(ast Asset) mgoAsset {
	mgoast := make([]mgoPoint, len(ast))
	for i, p := range ast {
		mgoast[i] = mgoPoint{
			T: p.T,
			V: p.V.String(),
		}
	}
	return mgoast
}

func ConvertMGOAsset(mgoast mgoAsset) Asset {
	ast := make([]Point, len(mgoast))
	for i, p := range mgoast {
		v, _ := new(big.Int).SetString(p.V, 10)
		ast[i] = Point{
			T: p.T,
			V: v,
		}
	}
	return ast
}

func ParseTxAndReceipt(obj bson.M) (tx ethapi.TxAndReceipt, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("mongo ParseTxAndReceipt fail, error: %v", r)
		}
	}()
	//tx.FsnTxInput = 
	tx.Tx = new(ethapi.RPCTransaction)
	for k, v := range obj["tx"].(bson.M) {
		switch k {
		case "blockHash":
			tx.Tx.BlockHash = common.HexToHash(v.(string))
		case "blockNumber":
			tx.Tx.BlockNumber = (*hexutil.Big)(big.NewInt(v.(int64)))
		case "from":
			tx.Tx.From = common.HexToAddress(v.(string))
		case "to":
			tx.Tx.To = new(common.Address)
			*tx.Tx.To = common.HexToAddress(v.(string))
		case "value":
			tx.Tx.Value = (*hexutil.Big)(big.NewInt(v.(int64)))
		}
	}
	receipt := obj["receipt"].(bson.M)
	tx.Receipt = make(map[string]interface{})
	tx.Receipt["transactionHash"] = common.HexToHash(receipt["transactionHash"].(string))
	tx.Receipt["fsnLogTopic"] = receipt["fsnLogTopic"].(string)
	tx.Receipt["fsnLogData"] = make(map[string]interface{})
	for k, v := range receipt["fsnLogData"].(bson.M) {
		switch k {
		case "AssetID":
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = v.(string)
		case "StartTime":
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = uint64(v.(float64))
		case "EndTime":
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = uint64(v.(float64))
		case "To":
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = common.HexToAddress(v.(string))
		case "Type":
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = int(v.(float64))
		case "LockType":
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = v.(string)
		case "Value":
			vr := new(big.Rat).SetFloat64(v.(float64))
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = new(big.Int).Quo(vr.Num(), vr.Denom())
		default:
			tx.Receipt["fsnLogData"].(map[string]interface{})[k] = v
		}
	}
	tx.Receipt["status"] = receipt["status"].(int)
	tx.ReceiptFound = obj["receiptFound"].(bool)
	return
}
