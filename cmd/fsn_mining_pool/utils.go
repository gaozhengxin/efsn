package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"time"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/common/hexutil"
	"github.com/FusionFoundation/efsn/core/types"
	"github.com/FusionFoundation/efsn/ethclient"
	"github.com/FusionFoundation/efsn/log"
)

func NewZeroTimer() *time.Timer {
	today := GetTodayZero()
	next := today.Add(time.Hour * 24)
	fmt.Printf("\nzero timer: next zero is %v\n\n", next)
	//timer := time.NewTimer(next.Sub(time.Now()))
	//fmt.Printf("!!!! timer is set: %v\n\n", next.Sub(time.Now()))
	timer := time.NewTimer(time.Second * 20) //测试
	return timer
}

func GetTodayZero() time.Time {
	return time.Now().Add(-1 * time.Hour * 12).Round(time.Hour * 24)
}

func GetRPCClient() *ethclient.Client {
	f := &NewRPCClient{}
	ret, err := Try(3, f, node)
	if err != nil {
		log.Warn("get rpc client failed")
		return nil
	}
	client := ret.(*ethclient.Client)
	return client
}

type Func interface {
	Do(...interface{}) (interface{}, error)
	Panic(error)
}

type NewRPCClient struct {}

func (f *NewRPCClient) Do (params ...interface{}) (interface{}, error) {
	return ethclient.Dial(params[0].(string))
}

func (f *NewRPCClient) Panic (err error) {
	log.Debug("create rpc client failed", "error", err)
}

func Try(trytimes int, f Func, params interface{}) (ret interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Try Func panic : %v", r)
		}
	}()
	for i := 1; i <= trytimes; i++ {
		ret, err = f.Do(params)
		if err != nil {
			log.Debug("Try " + fmt.Sprintf("%T", f) + " failed", "try time", i)
			f.Panic(err)
		} else {
			break
		}
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		return nil, fmt.Errorf("Do " + fmt.Sprintf("%T", f) + " failed")
	}
	return ret, nil
}

type CheckTx struct {
}

func (f *CheckTx) Do(params ...interface{}) (interface{}, error) {
	hash := params[0].(common.Hash)
	client := GetRPCClient()
	if client != nil {
		receipt, err := client.TransactionReceipt(context.Background(), hash)
		if err != nil {
			return nil, fmt.Errorf("get tx receipt failed: %v", err)
		}
		if receipt.Status != 1 {
			return nil, fmt.Errorf("tx is not confirmed")
		} else {
			return nil, nil
		}
	}
	return nil, fmt.Errorf("rpc client is nil")
}

func (f *CheckTx) Panic (err error) {
	log.Debug("check tx failed", "error", err)
}

var (
	GetCoinbaseCmd = `{"jsonrpc":"2.0","method":"eth_coinbase","id":10}`
	GetBalanceCmd = `{"jsonrpc":"2.0","method":"fsn_getBalance","params":["0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","%v","latest"],"id":"21"}`
	GetTimeLockBalanceCmd = `{"jsonrpc":"2.0","method":"fsn_getTimeLockBalance","params":["0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","%v","latest"],"id":"22"}`
	IsAutoBuyTicketCmd = `{"jsonrpc":"2.0","method":"fsntx_isAutoBuyTicket","params":["latest"],"id":30}`
	StartAutoBuyTicketCmd = `{"jsonrpc":"2.0","method":"miner_startAutoBuyTicket","params":[],"id":31}`
	StopAutoBuyTicketCmd = `{"jsonrpc":"2.0","method":"miner_stopAutoBuyTicket","params":[],"id":32}`
)

func GetCoinbase() common.Address {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("get coinbase failed", "error", r)
		}
	}()
	reqData := GetCoinbaseCmd
	res := PostJson(node, reqData)
	return common.HexToAddress(res.(string))
}

func StartAutoBuyTicket() bool {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("start auto buy ticket failed", "error", r)
		}
	}()
	reqData := StartAutoBuyTicketCmd
	PostJson(node, reqData)
	return IsAutoBuyTicket()
}

func StopAutoBuyTicket() bool {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("stop auto buy ticket failed", "error", r)
		}
	}()
	reqData := StopAutoBuyTicketCmd
	PostJson(node, reqData)
	return !IsAutoBuyTicket()
}

func IsAutoBuyTicket() bool {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("get IsAutoBuyTicket failed", "error", r)
		}
	}()
	reqData := IsAutoBuyTicketCmd
	res := PostJson(node, reqData)
	return res.(bool)
}

func GetBalance(addr common.Address) (ast *Asset) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("get balance failed", "error", r)
		}
	}()
	reqData := fmt.Sprintf(GetBalanceCmd, addr.Hex())
	res := PostJson(node, reqData)

	ast, err := NewAsset(big.NewInt(0), 0, 0)
	if err == nil {
		amt, _ := new(big.Int).SetString(res.(string), 10)
		ast, _ = NewAsset(amt, 0, 0)
	}
	return
}

func GetTimelockBalance(addr common.Address) (ast *Asset) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("get timelock balance failed", "error", r)
		}
	}()
	reqData := fmt.Sprintf(GetTimeLockBalanceCmd, addr.Hex())
	res := PostJson(node, reqData)

	ast, err := NewAsset(big.NewInt(0), 0, 0)
	if err == nil {
		for _, v := range res.(map[string]interface{})["Items"].([]interface{}) {
			amt, _ := new(big.Int).SetString(v.(map[string]interface{})["Value"].(string), 10)
			ast1, _ := NewAsset(amt, uint64(v.(map[string]interface{})["StartTime"].(float64)), uint64(v.(map[string]interface{})["EndTime"].(float64)))
			ast = ast.Add(ast1)
		}
	}
	ast.Reduce()
	ast.Align(uint64(GetTodayZero().Unix()))
	ast.Reduce()
	return
}

type JsonRes struct {
	Result interface{}
	Id interface{}
}

func PostJson(url, reqData string) interface{} {
	req := bytes.NewBuffer([]byte(reqData))
	resp, _ := http.Post(url, "application/json;charset=utf-8", req)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	jsonres := new(JsonRes)
	err := json.Unmarshal(body, jsonres)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil
	}
	return jsonres.Result
}

// a timelock has the form: {start time, end time, value}
// an asset that is a slice of time-value points:
// [{t_1,v_1},{t_2,v_2},...,{t_n,v_n}]
// is converted to a slice of timelocks:
// [{t_1,t_2,v_1},{t_2,t_3,v_2},...,{t_(n-1),t_n,v_(n-1)}]
// and if vn != 0, timelocks append {t_n,forever,v_n}
func sendAsset(from, to common.Address, asset *Asset, priv *ecdsa.PrivateKey) ([]common.Hash, error) {
	asset.Sort()
	asset.Reduce()
	if !asset.IsNonneg() {
		return nil, fmt.Errorf("sendAsset() failed: cannot send negative asset")
	}
	today := GetTodayZero().Unix()
	asset.Align(uint64(today))

	var argss []common.TimeLockArgs
	for i := 0; i < len(*asset); i++ {
		if (*asset)[i].V.Cmp(big.NewInt(0)) == 0 && len(*asset) > 1 {
			continue
		}
		args := &common.TimeLockArgs{}
		args.Init()
		args.To = to
		args.AssetID = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
		v := hexutil.Big(*(*asset)[i].V)
		args.Value = &v
		*(*uint64)(args.StartTime) = (*asset)[i].T
		if i != len(*asset) - 1 {
			t := hexutil.Uint64((*asset)[i+1].T)
			args.EndTime = &t
		}
		argss = append(argss, *args)
	}

	client := GetRPCClient()

	/*chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("cannot get chain id in SendFSN", "error", err)
	}*/

	chainID := big.NewInt(ChainID)

	gasLimit := uint64(80000)
	gasPrice := big.NewInt(1000000000)

	signer := types.NewEIP155Signer(chainID)
	addr := common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")

	var hs []common.Hash
	for _, args := range argss {
		nonce, err := client.PendingNonceAt(context.Background(), from)
		if err != nil {
			return nil, fmt.Errorf("cannot get nonce", "error", err)
		}
		var param common.FSNCallParam
		var funcData []byte
		if uint64(*args.EndTime) == common.TimeLockForever {
			funcData, _ = args.SendAssetArgs.ToData()
			param = common.FSNCallParam{Func: common.SendAssetFunc, Data: funcData}
		} else {
			funcData, _ = args.ToData(common.AssetToTimeLock)
			param = common.FSNCallParam{Func: common.TimeLockFunc, Data: funcData}
		}
		d, _ := param.ToBytes()
		data := hexutil.Bytes(d)
		if err != nil {
			return nil, fmt.Errorf("encode transaction data failed", "error", err)
		}
		tx := types.NewTransaction(nonce, addr, big.NewInt(0), gasLimit, gasPrice, data)

		// sign
		signedTx, _ := types.SignTx(tx, signer, priv)
		h := signedTx.Hash()

		err = client.SendTransaction(context.Background(), signedTx)
		if err != nil {
			log.Warn("send tx failed", "tx", tx, "error", err)
			continue
		}
		hs = append(hs, h)
	}

	var notconfirmed []common.Hash

	f := &CheckTx{}
	for _, h := range hs {
		log.Debug("check transaction", "hash", h.Hex())
		_, err := Try(10, f, h)
		if err != nil {
			notconfirmed = append(notconfirmed, h)
		}
	}

	if len(notconfirmed) > 0 {
		return hs, fmt.Errorf("%v transactions not confirmed: %+v", notconfirmed)
	}

	return hs, nil
}
