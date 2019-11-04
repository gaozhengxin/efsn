package sync

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/fatih/color"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/common/hexutil"
	"github.com/FusionFoundation/efsn/consensus/datong"
	"github.com/FusionFoundation/efsn/core/types"
	cnsl "github.com/FusionFoundation/efsn/console"
	"github.com/FusionFoundation/efsn/internal/ethapi"
	glog "github.com/FusionFoundation/efsn/log"
	"github.com/FusionFoundation/efsn/mongodb"
	"github.com/FusionFoundation/efsn/rlp"
	"github.com/FusionFoundation/efsn/rpc"
)

func InitSync() {
	color.NoColor = true
	ipcInit()
	mongodb.MongoInit()
	mongodb.Mongo = true
	glog.Root().SetHandler(glog.LvlFilterHandler(glog.LvlWarn, glog.StreamHandler(os.Stderr, glog.TerminalFormat(true))))
	//glog.Root().SetHandler(glog.LvlFilterHandler(glog.LvlDebug, glog.StreamHandler(os.Stderr, glog.TerminalFormat(true))))
}

var Myaddrs []string  = []string{""}

var Endpoint = ""

var MaxGoroutineNumber uint64 = 1000

var StartBlock uint64 = 0

var IpcDataDir string

var IpcDocRoot string

var Offset uint64 = 15

// 选择需要写入数据库的交易
var txFilter func(txrcp *ethapi.TxAndReceipt) bool = func(txrcp *ethapi.TxAndReceipt) bool {
	return true
}

func RegisterTxFilter(callback func(txrcp *ethapi.TxAndReceipt) bool) {
	txFilter = callback
}

var TxFilter2 = func(txrcp *ethapi.TxAndReceipt) (add bool) {
	defer func() {
		if r := recover(); r != nil {
			add = false
			return
		}
	}()
	// if tx is
	//   1. from myaddr
	//   2. to myaddr
	//   3. timelock tx to myaddr
	// then return true
	for _, Myaddr := range Myaddrs {
		if strings.EqualFold(txrcp.Receipt["from"].(string),Myaddr) || strings.EqualFold(txrcp.Receipt["to"].(string), Myaddr) || strings.EqualFold(txrcp.Receipt["fsnLogData"].(map[string]interface{})["To"].(string), Myaddr) {
			return true
		}
	}
	return false
}

func Sync() {
	ch := make(chan uint64, 1)

	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				height, e := blockNumber()
				if e == nil {
					if height > Offset {
						height = height - Offset
					} else {
						height = 1
					}
					ch <- height
				} else {
					log.Print(e.Error())
				}
			}
		}
	}()

	var head uint64 = StartBlock
	var total uint64 = 0
	h := head
	var ph uint64 = 0
	var v float64
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				v = float64(h - ph) * float64(12)
				fmt.Fprintf(os.Stdout, "%v/%v  %v\r", h, total, v)
				ph = h
			default:
				fmt.Fprintf(os.Stdout, "%v/%v  %v blocks/min\r", h, total, v)
			}
		}
	} ()

	go func() {
		// check blocks
		// check transactions
	}()

	for {
		select {
		case height := <-ch:
			total = height
			for head < height {
				/*
				block, _, e := getBlock(head + 1)
				if e != nil {
					glog.Warn("get block error", "block number", head, "error", e)
					head++
					continue
				}
				glog.Debug("sync", "block number", head, "block", fmt.Sprintf("%+v", block))
				//txs, e := getBlockTransactions(head)
				txs, e := getBlockTxRcpts(head)
				if e != nil {
					glog.Warn("get block transactions error", "block number", head, "error", e)
					head++
					continue
				}
				glog.Debug("sync", "block number", head, "transactions", fmt.Sprintf("%+v", txs))
				if mongodb.Mongo {
				//	mongodb.LightSync(block, txs)
				}
				head++
				*/

				// sync blocks in n routines
				// n = min[(height - head), MaxGoroutineNumber]
				d := height - head
				dda0 := (d + MaxGoroutineNumber)/(2 * d)
				mda0 := (d + MaxGoroutineNumber)/(2 * MaxGoroutineNumber)
				dda := dda0/(dda0 + mda0)
				mda := mda0/(dda0 + mda0)
				n := dda * d + mda * MaxGoroutineNumber
				//fmt.Printf("\n\n\n\n============\n  d = %v\n  dda = %v\n  mda = %v\n  n = %v\n============\n\n\n\n", d, dda, mda, n)
				//fmt.Printf("\n\n\n\n============\n  height = %v\n  head = %v\n  head + n = %v\n============\n\n\n\n", height, head, head + n)

				var wg sync.WaitGroup
				for i := head; i < head + n; i++ {
					wg.Add(int(1))
					go func(head uint64, h *uint64) {
						defer func() {
							wg.Done()
							if r := recover(); r != nil {
								glog.Warn(fmt.Sprintf("%v", r))
							}
						}()
						block, mb, e := getBlock(head + 1)
						if e != nil {
							glog.Warn("get block error", "block number", head, "error", e)
							return
						}
						glog.Debug("sync", "block number", head, "block", fmt.Sprintf("%+v", block))

						txs, e := getBlockTxRcpts(head, txFilter)
						if e != nil {
							glog.Warn("get block transactions error", "block number", head, "error", e)
							return
						}

						if mongodb.Mongo {
							mongodb.SyncTxs(txs)
							/*fmt.Printf("\n\nbuf len: %v\n\n", len(mongodb.GetTxBuf().Txs))
							if len(mongodb.GetTxBuf().Txs) >= 1000 {
								mongodb.TxBufPush()
							}*/

							CalculateReward(mb, txs)

							mongodb.AddBlock1(mb)
						}

						*h++

						hs := ""
						if l := len(txs); l > 0 {
							hs = txs[0].Tx.Hash.Hex()
							for i := 1; i < l; i++ {
								hs = hs + ", " + txs[i].Tx.Hash.Hex()
							}
						}
						glog.Debug("sync", "block number", head, "transactions", hs)
					}(i, &h)
				}
				wg.Wait()
				//mongodb.TxBufPush()
				head = head + n
			}
		}
	}
	mongodb.TxBufPush()
}

func StopSync() {
	mongodb.TxBufPush()
}

var (
	console *cnsl.Console
	printer *bytes.Buffer
	rw sync.RWMutex
)

var (
	NilConsoleErr error = fmt.Errorf("no ipc console")
	UnmarshalRPCBlockErr = func(w string) error {return fmt.Errorf("unmarshal rpc block error: %v", w)}
	RPCBlockHashErr error = fmt.Errorf("rcp block hash not match")
	NoTxErr error = fmt.Errorf("transaction not found")
	UnmarshalTxErr = func(w string) error {return fmt.Errorf("unmarshal transaction error: %v", w)}
	GetBlockTransactionsErr error = fmt.Errorf("get block transactions error")
)

var (
	eth_blockNumber = `eth.blockNumber`
	eth_getBlock = `JSON.stringify(eth.getBlock(%v))`
	eth_getTransactionFromBlock = `JSON.stringify(eth.getTransactionFromBlock("%v", %v))`
	eth_getTransaction = `JSON.stringify(eth.getTransaction("%v"))`
	eth_getBlockTransactionCount =`eth.getBlockTransactionCount(%v)`
	fsn_getTransactionAndReceipt = `JSON.stringify(fsn.getTransactionAndReceipt("%v"))`
)

func ipcInit() {
	fmt.Printf("!!!!! Endpoint is %v\n", Endpoint)
	client, err := rpc.Dial(Endpoint)
	if err != nil {
		log.Fatal(err)
		return
	}

	s := ""
	printer = bytes.NewBufferString(s)

	cfg := cnsl.Config{
		DataDir: IpcDataDir,
		DocRoot: IpcDocRoot,
		Client: client,
		Printer: printer,
	}

	console, err = cnsl.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
}

func getTransactionAndReceipt(hash string) (*ethapi.TxAndReceipt, error) {
	if console == nil {
		return nil, NilConsoleErr
	}
	rw.Lock()
	defer rw.Unlock()

	printer.Reset()
	code := fmt.Sprintf(fsn_getTransactionAndReceipt, hash)
	console.Evaluate(code)
	ret := printer.String()
	printer.Reset()

	ret = strings.TrimSuffix(ret, "\n")
	trimL := func (l rune) bool {return l != '{'}
	trimR := func (l rune) bool {return l != '}'}
	ret = strings.TrimLeftFunc(ret, trimL)
	ret = strings.TrimRightFunc(ret, trimR)
	ret = strings.Replace(ret, "\\", "", -1)

	txrcp := new(ethapi.TxAndReceipt)
	json.Unmarshal([]byte(ret), &txrcp)

	return txrcp, nil
}

func getBlockTransactions(number uint64) ([]*types.Transaction, error) {
	count, err := getBlockTransactionCount(number)
	if err != nil {
		return nil, GetBlockTransactionsErr
	}

	var txs []*types.Transaction
	for i := 0; i < count; i++ {
		tx, e := getTransactionFromBlock(number, i)
		if e != nil {
			if err != nil {
				err = fmt.Errorf("%v: tx; %v; %v", err.Error(), strconv.Itoa(i), e.Error())
			} else {
				err = fmt.Errorf("tx; %v; ", strconv.Itoa(i), e.Error())
			}
			continue
		}
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	return txs, err
}

func getBlockTxRcpts(number uint64, callback func(*ethapi.TxAndReceipt) bool) ([]*ethapi.TxAndReceipt, error) {
	count, err := getBlockTransactionCount(number)
	if err != nil {
		return nil, GetBlockTransactionsErr
	}

	var txs []*ethapi.TxAndReceipt
	for i := 0; i < count; i++ {
		tx, e := getTransactionFromBlock(number, i)
		if e != nil {
			if err != nil {
				err = fmt.Errorf("%v: tx; %v; %v", err.Error(), strconv.Itoa(i), e.Error())
			} else {
				err = fmt.Errorf("tx; %v; ", strconv.Itoa(i), e.Error())
			}
			continue
		}
		if tx != nil {
			s, e := getTransactionAndReceipt(tx.Hash().Hex())
			if e != nil {
				if err != nil {
					err = fmt.Errorf("%v: tx; %v; %v", err.Error(), strconv.Itoa(i), e.Error())
				} else {
					err = fmt.Errorf("tx; %v; ", strconv.Itoa(i), e.Error())
				}
				continue
			}
			/*if callback(s) {
				txs = append(txs, s)
			}*/
			txs = append(txs, s)
		}
	}
	return txs, err
}

func blockNumber() (uint64, error) {
	if console == nil {
		return 0, NilConsoleErr
	}
	rw.Lock()
	defer rw.Unlock()

	printer.Reset()
	console.Evaluate(eth_blockNumber)
	ret := printer.String()
	printer.Reset()

	ret = strings.TrimSuffix(ret, "\n")
	return strconv.ParseUint(ret, 10, 64)
}

func getBlockTransactionCount(number uint64) (int, error) {
	if console == nil {
		return 0, NilConsoleErr
	}
	rw.Lock()
	defer rw.Unlock()

	printer.Reset()
	code := fmt.Sprintf(eth_getBlockTransactionCount, number)
	console.Evaluate(code)
	ret := printer.String()
	printer.Reset()

	ret = strings.TrimSuffix(ret, "\n")
	return strconv.Atoi(ret)
}

func getBlock(number uint64) (_ *types.Block, _ *mgoBlock, err error) {
	if console == nil {
		return nil, nil, NilConsoleErr
	}
	rw.Lock()
	defer rw.Unlock()

	printer.Reset()
	code := fmt.Sprintf(eth_getBlock, number)
	console.Evaluate(code)
	ret := printer.String()
	printer.Reset()

	trimL := func (l rune) bool {return l != '{'}
	trimR := func (l rune) bool {return l != '}'}
	ret = strings.TrimLeftFunc(ret, trimL)
	ret = strings.TrimRightFunc(ret, trimR)
	ret = strings.Replace(ret, "\\", "", -1)

	rpcblk := new(RPCBlock)
	err = json.Unmarshal([]byte(ret), rpcblk)
	if err != nil {
		err = UnmarshalRPCBlockErr(err.Error() + "    " + ret)
		return nil, nil, err
	}

	blk1, e1 := rpcblk.ToBlock()
	if e1 != nil {
		return nil, nil, e1
	}
	blk2, e2 := rpcblk.ToMgoBlock()
	if e2 != nil {
		return nil, nil, e2
	}
	return blk1, blk2, nil
}

func getTransactionFromBlock(block uint64, number int) (*types.Transaction, error) {
	if console == nil {
		return nil, NilConsoleErr
	}
	rw.Lock()
	defer rw.Unlock()

	printer.Reset()
	code := fmt.Sprintf(eth_getTransactionFromBlock, block, number)
	console.Evaluate(code)
	ret := printer.String()
	printer.Reset()

	trimL := func (l rune) bool {return l != '{'}
	trimR := func (l rune) bool {return l != '}'}
	ret = strings.TrimLeftFunc(ret, trimL)
	ret = strings.TrimRightFunc(ret, trimR)
	ret = strings.Replace(ret, "\\", "", -1)
	return UnmarshalTx(ret)
}

func getTransaction(hash string) (*types.Transaction, error) {
	if console == nil {
		return  nil, NilConsoleErr
	}
	rw.Lock()
	defer rw.Unlock()

	printer.Reset()
	code := fmt.Sprintf(eth_getTransaction, hash)

	console.Evaluate(code)
	ret := printer.String()
	printer.Reset()

	trimL := func (l rune) bool {return l != '{'}
	trimR := func (l rune) bool {return l != '}'}
	ret = strings.TrimLeftFunc(ret, trimL)
	ret = strings.TrimRightFunc(ret, trimR)
	ret = strings.Replace(ret, "\\", "", -1)
	return UnmarshalTx(ret)
}

func CalculateReward(mb *mgoBlock, txs []*ethapi.TxAndReceipt) {
	defer func() {
		if r := recover(); r != nil {
			glog.Warn("CalculateReward() failed", "error", r)
		}
	}()
	// block creation reward
	reward := datong.CalcRewards(new(big.Int).SetUint64(mb.Number))
	gasUses := make(map[common.Hash]*big.Int)
	for _, tx := range txs {
		gasUsed := big.NewInt(0)
		if tx.Receipt["gasUsed"] != nil {
			gasUsed, _ = new(big.Int).SetString(tx.Receipt["gasUsed"].(string), 0)
		}
		gasUses[tx.Tx.Hash] = gasUsed
	}
	for _, tx := range txs {
		if gasUsed, ok := gasUses[tx.Tx.Hash]; ok {
			gasReward := new(big.Int).Mul(tx.Tx.GasPrice.ToInt(), gasUsed)
			if gasReward.Sign() > 0 {
				// transaction gas reward
				reward.Add(reward, gasReward)
			}
		}
		if common.IsFsnCall(tx.Tx.To) {
			fsnCallParam := &common.FSNCallParam{}
			if tx.Receipt["logs"].([]interface{})[0].(map[string]interface{})["data"] != nil {
				data, _ := hex.DecodeString(tx.Receipt["logs"].([]interface{})[0].(map[string]interface{})["data"].(string))
				rlp.DecodeBytes(data, fsnCallParam)
				feeReward := common.GetFsnCallFee(tx.Tx.To, fsnCallParam.Func)
				if feeReward.Sign() > 0 {
					// transaction fee reward
					reward.Add(reward, feeReward)
				}
			}
		}
	}

	mb.Reward = reward.String()
}

func UnmarshalTx(input string) (tx *types.Transaction, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = UnmarshalTxErr(fmt.Sprintf("%+v", r))
		}
	}()

	if input == "" {
		return nil, NoTxErr
	}

	tmp := make(map[string]interface{})
	err = json.Unmarshal([]byte(input), &tmp)
	if err != nil {
		return nil, UnmarshalTxErr(err.Error() + "    " + input)
	}
	if tmp["gas"] != nil {
		gas := tmp["gas"].(float64)
		tmp["gas"] = hexutil.EncodeUint64(uint64(gas))
	}
	if tmp["gasPrice"] != nil {
		gasPrice := tmp["gasPrice"].(string)
		pbig, _ := new(big.Int).SetString(gasPrice, 10)
		tmp["gasPrice"] = hexutil.EncodeBig(pbig)
	}
	if tmp["nonce"] != nil {
		nonce := tmp["nonce"].(float64)
		tmp["nonce"] = hexutil.EncodeUint64(uint64(nonce))
	}
	if tmp["value"] != nil {
		value := tmp["value"].(string)
		vf, _ := strconv.ParseFloat(value, 64)
		vs := strconv.FormatFloat(vf, 'f', -1, 64)
		vbig, _ := new(big.Int).SetString(vs, 10)
		tmp["value"] = hexutil.EncodeBig(vbig)
	}
	tmp2, err := json.Marshal(tmp)
	if err != nil {
		return nil, UnmarshalTxErr(err.Error() + "    " + input)
	}

	input = string(tmp2)

	tx = new(types.Transaction)
	err = tx.UnmarshalJSON([]byte(input))
	if err != nil {
		return nil, UnmarshalTxErr(err.Error() + "    " + input)
	}
	return tx, nil
}

func (rpcblk *RPCBlock) ToBlock () (blk *types.Block, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = UnmarshalRPCBlockErr(fmt.Sprintf("%+v", r))
		}
	}()

	header := &types.Header{
		Number: big.NewInt(rpcblk.Number),
		ParentHash: rpcblk.ParentHash,
		MixDigest: rpcblk.MixHash,
		UncleHash: rpcblk.Sha3Uncles,
		Bloom: rpcblk.LogsBloom,
		Root: rpcblk.StateRoot,
		Coinbase: rpcblk.Miner,
		Extra: []byte(*rpcblk.ExtraData),
		GasLimit: rpcblk.GasLimit,
		GasUsed: rpcblk.GasUsed,
		Time: big.NewInt(rpcblk.Timestamp),
		TxHash: rpcblk.TransactionsRoot,
		ReceiptHash: rpcblk.ReceiptsRoot,
	}

	difficulty, ok := new(big.Int).SetString(rpcblk.Difficulty, 10)
	if !ok {
		err = UnmarshalRPCBlockErr("parse difficulty number error")
	}
	header.Difficulty = difficulty

	nonce := new(types.BlockNonce)
	err = nonce.UnmarshalText([]byte(rpcblk.Nonce))
	if err != nil {
		err = UnmarshalRPCBlockErr("nonce unmarshal text: " + err.Error())
		return
	}
	header.Nonce = *nonce

	//blk = types.NewBlock(header, nil, nil, nil)
	blk = types.NewBlockWithHeader(header)

	if hash := blk.Hash(); hash != rpcblk.Hash {
		//fmt.Println(hash.Hex())
		//fmt.Println(rpcblk.Hash.Hex())
		err = RPCBlockHashErr
	}
	return
}

func (rpcblk *RPCBlock) ToMgoBlock () (blk *mgoBlock, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = UnmarshalRPCBlockErr(fmt.Sprintf("%+v", r))
		}
	}()

	mb := new(mgoBlock)
	mb.Key = rpcblk.Hash.Hex()
	mb.Number = uint64(rpcblk.Number)
	mb.Hash = rpcblk.Hash.Hex()
	mb.ParentHash = rpcblk.ParentHash.Hex()
	mb.Nonce = rpcblk.Nonce
	mb.Sha3Uncles = rpcblk.Sha3Uncles.Hex()
	mb.LogsBloom = string(rpcblk.LogsBloom[:])
	mb.TransactionsRoot = rpcblk.StateRoot.Hex()
	mb.StateRoot = rpcblk.StateRoot.Hex()
	mb.Miner = rpcblk.Miner.Hex()
	mb.Difficulty, _ = strconv.ParseUint(rpcblk.Difficulty, 10, 64)
	//mb.TotalDifficulty
	mb.Size = float64(rpcblk.Size)
	mb.ExtraData = string(*rpcblk.ExtraData)
	mb.GasLimit = rpcblk.GasLimit
	mb.GasUsed = rpcblk.GasUsed
	mb.Timestamp = uint64(rpcblk.Timestamp)
	//mb.Uncles
	mb.Txns = len(rpcblk.Transactions)

	//mb.AvgGasprice
	//mb.Reward = rpcblk.
	mb.ReceiptHash = rpcblk.ReceiptsRoot.Hex()
	//mb.BlockTime
	return mb, nil
}

type RPCBlock struct {
	Number            int64
	Hash              common.Hash
	ParentHash        common.Hash
	Nonce             string
	MixHash           common.Hash
	Sha3Uncles        common.Hash
	LogsBloom         types.Bloom
	StateRoot         common.Hash
	Miner             common.Address
	Difficulty        string
	ExtraData         *hexutil.Bytes
	Size              uint64
	GasLimit          uint64
	GasUsed           uint64
	Timestamp         int64
	TransactionsRoot  common.Hash
	ReceiptsRoot      common.Hash
	Transactions      []common.Hash
	Uncles            []common.Hash
}

type mgoBlock struct {
	Key              string          `bson:"_id"`
	Number           uint64          `bson:"number"`
	Hash             string          `bson:"hash"`
	ParentHash       string          `bson:"parentHash"`
	Nonce            string          `bson:"nonce"`
	Sha3Uncles       string          `bson:"sha3Uncles"`
	LogsBloom        string          `bson:"logsBloom"`
	TransactionsRoot string          `bson:"transactionsRoot"`
	StateRoot        string          `bson:"stateRoot"`
	Miner            string          `bson:"miner"`
	Difficulty       uint64          `bson:"difficulty"`
	TotalDifficulty  uint64          `bson:"totalDifficulty"`
	Size             float64         `bson:"size"`
	ExtraData        string          `bson:"extraData"`
	GasLimit         uint64          `bson:"gasLimit"`
	GasUsed          uint64          `bson:"gasUsed"`
	Timestamp        uint64          `bson:"timestamp"`
	Uncles           []*types.Header `bson:"uncles"`
	Txns             int             `bson:"txns"`
	AvgGasprice      string          `bson:"avgGasprice"`
	Reward           string          `bson:"reward"`

	ReceiptHash string `bson:"receiptHash"`
	BlockTime   uint64 `bson:"blockTime"`
}
