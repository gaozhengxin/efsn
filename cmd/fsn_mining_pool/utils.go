package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
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
	ret, err := Try(3, f, url)
	if err != nil {
		log.Warn("get rpc client failed")
		return nil
	}
	client := ret.(*ethclient.Client)
	return client
}

type Func interface {
	Name() string
	Do(...interface{}) (interface{}, error)
	Panic(error)
}

type NewRPCClient struct {}

func (f *NewRPCClient) Name() string {return "create rpc client getBalance"}

func (f *NewRPCClient) Do (params ...interface{}) (interface{}, error) {
	return ethclient.Dial(params[0].(string))
}

func (f *NewRPCClient) Panic (err error) {
	log.Debug("create rpc client getBalance failed", "error", err)
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
			log.Debug("Try " + f.Name() + "failed", "try time", i)
			f.Panic(err)
		} else {
			break
		}
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		return nil, fmt.Errorf("Do " + f.Name() + "failed")
	}
	return ret, nil
}

type CheckTx struct {
}

func (f *CheckTx) Name() string {return "check tx"}

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

	//chainID := big.NewInt(32659)
	//chainID := big.NewInt(46688)
	chainID := big.NewInt(55555)

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
