package vm

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/core/types"
	"github.com/FusionFoundation/efsn/crypto"
	"github.com/FusionFoundation/efsn/crypto/sha3"
	"github.com/FusionFoundation/efsn/rlp"
)

type FSNContractBase struct {
	input []byte
}

type FSNContract struct {
	FSNContractBase
	evm    *EVM
	caller common.Address
}

func NewFSNContractBase(input []byte) *FSNContractBase {
	return &FSNContractBase{
		input: input,
	}
}

func NewFSNContract(input []byte, caller common.Address, evm *EVM) *FSNContract {
	return &FSNContract{
		FSNContractBase: FSNContractBase{
			input: input,
		},
		evm:    evm,
		caller: caller,
	}
}

func processError(caller common.Address, err error) ([]byte, error) {
	common.DebugInfo("RunFSNContract failed", "caller", caller, "err", err)
	return ErrToStorageBytes(err), err
}

func RunFSNContract(input []byte, caller common.Address, evm *EVM, value *big.Int) (bs []byte, err error) {
	if !common.IsFsnContractEnabled(evm.BlockNumber) {
		err = fmt.Errorf("FsnContract is not enabled")
		return processError(caller, err)
	}
	if value.Sign() != 0 {
		err = fmt.Errorf("FsnContract address is non-paybale")
		return processError(caller, err)
	}
	fsnContract := NewFSNContract(input, caller, evm)
	bs, err = fsnContract.Run()
	if err != nil {
		return processError(caller, err)
	}
	common.DebugInfo("RunFSNContract finished", "caller", caller, "input", input, "result", bs, "err", err)
	return bs, err
}

func RequiredFee(input []byte) *big.Int {
	return common.GetFsnCallFee(common.GetFsnFuncType(input))
}

func FSNContractQueryBalance(state StateDB, input []byte) ([]byte, error) {
	switch common.GetFsnFuncHash(input) {
	case common.GetAllBalancesFH:
		return FSNContractGetAllBalances(state, input)
	case common.GetBalanceFH:
		return FSNContractGetBalance(state, input)
	}
	return nil, fmt.Errorf("Only query balance is allowed")
}

func (c *FSNContract) Run() ([]byte, error) {
	funcHash := common.GetFsnFuncHash(c.input)
	switch funcHash {
	case common.SendAssetFH, common.USANsendAssetFH:
		return c.sendAsset()
	case common.TimeLockToAssetFH, common.USANtimeLockToAssetFH:
		return c.timeLockToAsset()
	case common.TimeLockToTimeLockFH, common.USANtimeLockToTimeLockFH:
		return c.timeLockToTimeLock()
	case common.AssetToTimeLockFH, common.USANassetToTimeLockFH:
		return c.assetToTimeLock()
	case common.BuyTicketFH:
		return c.buyTicket()
	case common.DecAssetFH:
		return c.decAsset()
	case common.GenAssetFH:
		return c.genAsset()
	case common.GenNotationFH:
		return c.genNotation()
	case common.IncAssetFH:
		return c.incAsset()
	case common.MakeMultiSwapFH, common.MakeMultiSwap2FH:
		return c.makeMultiSwap()
	case common.MakeSwapFH:
		return c.makeSwap()
	case common.RecallMultiSwapFH:
		return c.recallMultiSwap()
	case common.RecallSwapFH:
		return c.recallSwap()
	case common.TakeMultiSwapFH:
		return c.takeMultiSwap()
	case common.TakeSwapFH:
		return c.takeSwap()
	case common.GetAddressByNotationFH:
		return c.getAddressByNotation()
	case common.GetAllBalancesFH:
		return c.getAllBalances()
	case common.GetAllTimeLockBalancesFH:
		return c.getAllTimeLockBalances()
	case common.GetAssetFH:
		return c.getAsset()
	case common.GetBalanceFH:
		return c.getBalance()
	case common.GetMultiSwapFH:
		return c.getMultiSwap()
	case common.GetNotationFH:
		return c.getNotation()
	case common.GetSwapFH:
		return c.getSwap()
	case common.GetTimeLockBalanceFH:
		return c.getTimeLockBalance()
	case common.TicketPriceFH:
		return c.ticketPrice()
	case common.TotalNumberOfTicketsFH:
		return c.totalNumberOfTickets()
	case common.TotalNumberOfTicketsByAddressFH:
		return c.totalNumberOfTicketsByAddress()
	case common.GetStakeInfoFH:
		return c.getStakeInfo()
	case common.AllTicketsFH:
		return c.getAllTickets()
	case common.AllTicketsByAddressFH:
		return c.getAllTicketsByAddress()
	}

	return nil, fmt.Errorf("Unsupported func hash %s", funcHash)
}

// ensure uniqe ID generated
func (c *FSNContract) getIDHash() (hash common.Hash) {
	from := c.getFrom()
	nonce := c.evm.StateDB.GetNonce(from)

	hasher := sha3.NewKeccak256()
	rlp.Encode(hasher, []interface{}{
		from,
		nonce,
		c.input,
		c.evm.GasLimit,
		c.evm.GasPrice,
		c.evm.Coinbase,
		c.evm.BlockNumber,
		c.evm.Time,
		c.evm.Difficulty,
	})
	hasher.Sum(hash[:0])
	return hash
}

func getBigIntn(data []byte, offset int, n int) *big.Int {
	if offset+n > len(data) {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(data[offset : offset+n])
}

func getBigInt(data []byte, offset int) *big.Int {
	return getBigIntn(data, offset, 32)
}

func getInt(data []byte, offset int) int {
	return int(getBigInt(data, offset).Int64())
}

func (c *FSNContractBase) getBigInt(offset int) *big.Int {
	return getBigInt(c.input, offset)
}

// pos index start from 1, only used for argument retrieve
func (c *FSNContractBase) getPosOffset(pos int) int {
	return 4 + (pos-1)*32
}

func (c *FSNContractBase) getBigIntParam(pos int) *big.Int {
	return c.getBigInt(c.getPosOffset(pos))
}

func (c *FSNContractBase) getBoolParam(pos int) bool {
	return c.getBigIntParam(pos).Sign() != 0
}

func (c *FSNContractBase) getUint64Param(pos int) uint64 {
	return c.getBigIntParam(pos).Uint64()
}

func (c *FSNContractBase) getIntParam(pos int) int {
	return int(c.getBigIntParam(pos).Int64())
}

func getTimeOrDefault(bigVal *big.Int, defVal uint64) uint64 {
	if bigVal.Cmp(new(big.Int).SetUint64(common.TimeLockForever)) > 0 {
		return common.TimeLockForever
	}
	if bigVal.Sign() == 0 {
		return defVal
	}
	return bigVal.Uint64()
}

func getTimeParam(data []byte, offset int, defVal uint64) uint64 {
	return getTimeOrDefault(getBigInt(data, offset), defVal)
}

func (c *FSNContractBase) getStartTimeParam(pos int) uint64 {
	return getTimeParam(c.input, c.getPosOffset(pos), common.TimeLockNow)
}

func (c *FSNContractBase) getEndTimeParam(pos int) uint64 {
	return getTimeParam(c.input, c.getPosOffset(pos), common.TimeLockForever)
}

func getTimeArrayParam(data []byte, posOffset int, defVal uint64) []uint64 {
	offset := 4 + getInt(data, posOffset)
	length := getInt(data, offset)
	if 4+offset+32+length*32 > len(data) {
		return nil
	}
	array := make([]uint64, length)
	for i := 0; i < length; i++ {
		offset += 32
		array[i] = getTimeParam(data, offset, defVal)
	}
	return array
}

func (c *FSNContractBase) getStartTimeArrayParam(pos int) []uint64 {
	return getTimeArrayParam(c.input, c.getPosOffset(pos), common.TimeLockNow)
}

func (c *FSNContractBase) getEndTimeArrayParam(pos int) []uint64 {
	return getTimeArrayParam(c.input, c.getPosOffset(pos), common.TimeLockForever)
}

func (c *FSNContractBase) getBigIntArrayParam(pos int) []*big.Int {
	data := c.input
	offset := 4 + c.getIntParam(pos)
	length := getInt(data, offset)
	if 4+offset+32+length*32 > len(data) {
		return nil
	}
	array := make([]*big.Int, length)
	for i := 0; i < length; i++ {
		offset += 32
		array[i] = getBigInt(data, offset)
	}
	return array
}

func (c *FSNContractBase) getStringParam(pos int) string {
	data := c.input
	offset := 4 + c.getIntParam(pos)
	length := getInt(data, offset)
	offset += 32
	if offset+length > len(data) {
		return ""
	}
	return string(data[offset : offset+length])
}

func (c *FSNContractBase) getHashParam(pos int) common.Hash {
	return common.BigToHash(c.getBigIntParam(pos))
}

func (c *FSNContractBase) getHashArrayParam(pos int) []common.Hash {
	data := c.input
	offset := 4 + c.getIntParam(pos)
	length := getInt(data, offset)
	if 4+offset+32+length*32 > len(data) {
		return nil
	}
	array := make([]common.Hash, length)
	for i := 0; i < length; i++ {
		offset += 32
		array[i] = common.BigToHash(getBigInt(data, offset))
	}
	return array
}

func (c *FSNContractBase) getAddressParam(pos int) common.Address {
	return common.BigToAddress(c.getBigIntParam(pos))
}

func (c *FSNContractBase) getAddressArrayParam(pos int) []common.Address {
	data := c.input
	offset := 4 + c.getIntParam(pos)
	length := getInt(data, offset)
	if 4+offset+32+length*32 > len(data) {
		return nil
	}
	array := make([]common.Address, length)
	for i := 0; i < length; i++ {
		offset += 32
		array[i] = common.BigToAddress(getBigInt(data, offset))
	}
	return array
}

func (c *FSNContractBase) GetGenAssetParam() (*common.GenAssetParam, error) {
	name := c.getStringParam(1)
	symbol := c.getStringParam(2)
	decimals := uint8(c.getUint64Param(3))
	total := c.getBigIntParam(4)
	canChange := c.getBoolParam(5)
	description := c.getStringParam(6)

	return &common.GenAssetParam{
		Name:        name,
		Symbol:      symbol,
		Decimals:    decimals,
		Total:       total,
		CanChange:   canChange,
		Description: description,
	}, nil
}

func (c *FSNContractBase) GetBuyTicketParam(defstart uint64) (*common.BuyTicketParam, error) {
	start := getTimeParam(c.input, c.getPosOffset(1), defstart)
	end := getTimeParam(c.input, c.getPosOffset(2), start+30*24*3600)
	return &common.BuyTicketParam{
		Start: start,
		End:   end,
	}, nil
}

func (c *FSNContractBase) GetSendAssetParam(state StateDB) (*common.SendAssetParam, error) {
	asset := c.getHashParam(1)
	var to common.Address
	if common.IsUsingUSAN(c.input) {
		notation := c.getUint64Param(2)
		addr, err := state.GetAddressByNotation(notation)
		if err != nil {
			return nil, err
		}
		to = addr
	} else {
		to = c.getAddressParam(2)
	}
	value := c.getBigIntParam(3)
	return &common.SendAssetParam{
		AssetID: asset,
		To:      to,
		Value:   value,
	}, nil
}

func (c *FSNContractBase) GetAssetValueChangeExParam(from common.Address) (*common.AssetValueChangeExParam, error) {
	var (
		asset common.Hash
		to    common.Address
		value *big.Int
	)
	isInc := common.IsIncAsset(c.input)
	if isInc {
		asset = c.getHashParam(1)
		to = c.getAddressParam(2)
		value = c.getBigIntParam(3)
	} else {
		asset = c.getHashParam(1)
		value = c.getBigIntParam(2)
		to = from
	}
	return &common.AssetValueChangeExParam{
		AssetID: asset,
		To:      to,
		Value:   value,
		IsInc:   isInc,
	}, nil
}

func (c *FSNContractBase) GetTimeLockParam(state StateDB) (*common.TimeLockParam, error) {
	var (
		asset common.Hash
		to    common.Address
		start uint64
		end   uint64
		value *big.Int
	)

	asset = c.getHashParam(1)
	if common.IsUsingUSAN(c.input) {
		notation := c.getUint64Param(2)
		addr, err := state.GetAddressByNotation(notation)
		if err != nil {
			return nil, err
		}
		to = addr
	} else {
		to = c.getAddressParam(2)
	}

	lockType, err := common.GetTimeLockType(c.input)
	if err != nil {
		return nil, err
	}

	switch lockType {
	case common.AssetToTimeLock, common.TimeLockToTimeLock:
		start = c.getStartTimeParam(3)
		end = c.getEndTimeParam(4)
		value = c.getBigIntParam(5)
	case common.TimeLockToAsset:
		value = c.getBigIntParam(3)
		start = common.TimeLockNow
		end = common.TimeLockForever
	default:
		return nil, fmt.Errorf("unknown timelock type")
	}
	return &common.TimeLockParam{
		Type:      lockType,
		AssetID:   asset,
		To:        to,
		StartTime: start,
		EndTime:   end,
		Value:     value,
	}, nil
}

func (c *FSNContractBase) GetMakeSwapParam() (*common.MakeSwapParam, error) {
	fromAssetID := c.getHashParam(1)
	fromStartTime := c.getStartTimeParam(2)
	fromEndTime := c.getEndTimeParam(3)
	minFromAmount := c.getBigIntParam(4)
	toAssetID := c.getHashParam(5)
	toStartTime := c.getStartTimeParam(6)
	toEndTime := c.getEndTimeParam(7)
	minToAmount := c.getBigIntParam(8)
	swapSize := c.getBigIntParam(9)
	targets := c.getAddressArrayParam(10)
	description := c.getStringParam(11)

	return &common.MakeSwapParam{
		FromAssetID:   fromAssetID,
		FromStartTime: fromStartTime,
		FromEndTime:   fromEndTime,
		MinFromAmount: minFromAmount,
		ToAssetID:     toAssetID,
		ToStartTime:   toStartTime,
		ToEndTime:     toEndTime,
		MinToAmount:   minToAmount,
		SwapSize:      swapSize,
		Targes:        targets,
		Description:   description,
	}, nil
}

func (c *FSNContractBase) GetMakeMultiSwapParam() (*common.MakeMultiSwapParam, error) {
	if common.IsMakeMultiSwap2(c.input) {
		return c.GetMakeMultiSwap2Param()
	}

	fromAssetID := c.getHashArrayParam(1)
	fromStartTime := c.getStartTimeArrayParam(2)
	fromEndTime := c.getEndTimeArrayParam(3)
	minFromAmount := c.getBigIntArrayParam(4)
	toAssetID := c.getHashArrayParam(5)
	toStartTime := c.getStartTimeArrayParam(6)
	toEndTime := c.getEndTimeArrayParam(7)
	minToAmount := c.getBigIntArrayParam(8)
	swapSize := c.getBigIntParam(9)
	targets := c.getAddressArrayParam(10)
	description := c.getStringParam(11)

	return &common.MakeMultiSwapParam{
		FromAssetID:   fromAssetID,
		FromStartTime: fromStartTime,
		FromEndTime:   fromEndTime,
		MinFromAmount: minFromAmount,
		ToAssetID:     toAssetID,
		ToStartTime:   toStartTime,
		ToEndTime:     toEndTime,
		MinToAmount:   minToAmount,
		SwapSize:      swapSize,
		Targes:        targets,
		Description:   description,
	}, nil
}

// fromAmounts,toAmounts have the following format (3 big int for each asset)
// startTime1,endTime1,minAmount1,startTime2,endTime2,minAmount2,...
func (c *FSNContractBase) GetMakeMultiSwap2Param() (*common.MakeMultiSwapParam, error) {
	fromAssetID := c.getHashArrayParam(1)
	fromAmounts := c.getBigIntArrayParam(2)
	toAssetID := c.getHashArrayParam(3)
	toAmounts := c.getBigIntArrayParam(4)
	swapSize := c.getBigIntParam(5)
	targets := c.getAddressArrayParam(6)
	description := c.getStringParam(7)

	numFromAssetID := len(fromAssetID)
	numFromAmount := len(fromAmounts)
	numToAssetID := len(toAssetID)
	numToAmount := len(toAmounts)

	if numFromAmount != numFromAssetID*3 {
		return nil, fmt.Errorf("wrong number of fromAmounts")
	}

	if numToAmount != numToAssetID*3 {
		return nil, fmt.Errorf("wrong number of toAmounts")
	}

	var (
		fromStartTime []uint64
		fromEndTime   []uint64
		minFromAmount []*big.Int
		toStartTime   []uint64
		toEndTime     []uint64
		minToAmount   []*big.Int
	)

	defStartTime := common.TimeLockNow
	defEndTime := common.TimeLockForever

	for i := 0; i < numFromAmount; i += 3 {
		fromStartTime = append(fromStartTime, getTimeOrDefault(fromAmounts[i], defStartTime))
		fromEndTime = append(fromEndTime, getTimeOrDefault(fromAmounts[i+1], defEndTime))
		minFromAmount = append(minFromAmount, fromAmounts[i+2])
	}

	for i := 0; i < numToAmount; i += 3 {
		toStartTime = append(toStartTime, getTimeOrDefault(toAmounts[i], defStartTime))
		toEndTime = append(toEndTime, getTimeOrDefault(toAmounts[i+1], defEndTime))
		minToAmount = append(minToAmount, toAmounts[i+2])
	}

	return &common.MakeMultiSwapParam{
		FromAssetID:   fromAssetID,
		FromStartTime: fromStartTime,
		FromEndTime:   fromEndTime,
		MinFromAmount: minFromAmount,
		ToAssetID:     toAssetID,
		ToStartTime:   toStartTime,
		ToEndTime:     toEndTime,
		MinToAmount:   minToAmount,
		SwapSize:      swapSize,
		Targes:        targets,
		Description:   description,
	}, nil
}

func (c *FSNContractBase) GetRecallSwapParam() (*common.RecallSwapParam, error) {
	swapID := c.getHashParam(1)
	return &common.RecallSwapParam{
		SwapID: swapID,
	}, nil
}

func (c *FSNContractBase) GetRecallMultiSwapParam() (*common.RecallMultiSwapParam, error) {
	swapID := c.getHashParam(1)
	return &common.RecallMultiSwapParam{
		SwapID: swapID,
	}, nil
}

func (c *FSNContractBase) GetTakeSwapParam() (*common.TakeSwapParam, error) {
	swapID := c.getHashParam(1)
	size := c.getBigIntParam(2)
	return &common.TakeSwapParam{
		SwapID: swapID,
		Size:   size,
	}, nil
}

func (c *FSNContractBase) GetTakeMultiSwapParam() (*common.TakeMultiSwapParam, error) {
	swapID := c.getHashParam(1)
	size := c.getBigIntParam(2)
	return &common.TakeMultiSwapParam{
		SwapID: swapID,
		Size:   size,
	}, nil
}

func (c *FSNContract) getFrom() common.Address {
	return c.caller
}

func (c *FSNContract) getParentTime() uint64 {
	return c.evm.ParentTime.Uint64()
}

func (c *FSNContract) buyTicket() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.evm.Time.Uint64()

	buyTicketParam, err := c.GetBuyTicketParam(timestamp)
	if err != nil {
		return nil, err
	}

	paramData, _ := buyTicketParam.ToBytes() // compatible with FsnCall impl

	// adjust to parent time to check end time param
	adjust := new(big.Int).Sub(c.evm.ParentTime, c.evm.Time).Int64()
	if err := buyTicketParam.Check(height, timestamp, adjust); err != nil {
		c.addLog(common.BuyTicketFunc, paramData, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	hash := c.evm.GetHash(height.Uint64() - 1)
	id := crypto.Keccak256Hash(from[:], hash[:])

	if state.IsTicketExist(id) {
		err = fmt.Errorf("Ticket already exist")
		c.addLog(common.BuyTicketFunc, paramData, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	start := buyTicketParam.Start
	end := buyTicketParam.End
	value := common.TicketPrice(height)
	var needValue *common.TimeLock

	needValue = common.NewTimeLock(&common.TimeLockItem{
		StartTime: common.MaxUint64(start, timestamp),
		EndTime:   end,
		Value:     value,
	})
	if err := needValue.IsValid(); err != nil {
		c.addLog(common.BuyTicketFunc, paramData, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	ticket := common.Ticket{
		Owner: from,
		TicketBody: common.TicketBody{
			ID:         id,
			Height:     height.Uint64(),
			StartTime:  start,
			ExpireTime: end,
		},
	}

	useAsset := false
	if state.GetTimeLockBalance(common.SystemAssetID, from).Cmp(needValue) < 0 {
		if state.GetBalance(common.SystemAssetID, from).Cmp(value) < 0 {
			err = fmt.Errorf("not enough time lock or asset balance")
			c.addLog(common.BuyTicketFunc, paramData, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}
		useAsset = true
	}

	if useAsset {
		state.SubBalance(from, common.SystemAssetID, value)

		totalValue := common.NewTimeLock(&common.TimeLockItem{
			StartTime: timestamp,
			EndTime:   common.TimeLockForever,
			Value:     value,
		})
		surplusValue := new(common.TimeLock).Sub(totalValue, needValue)
		if !surplusValue.IsEmpty() {
			state.AddTimeLockBalance(from, common.SystemAssetID, surplusValue, height, timestamp)
		}

	} else {
		state.SubTimeLockBalance(from, common.SystemAssetID, needValue, height, timestamp)
	}

	if err := state.AddTicket(ticket); err != nil {
		c.addLog(common.BuyTicketFunc, paramData, common.NewKeyValue("Error", "unable to add ticket"))
		return nil, err
	}
	c.addLog(common.BuyTicketFunc, paramData, common.NewKeyValue("TicketID", ticket.ID), common.NewKeyValue("TicketOwner", ticket.Owner))
	return OkToStorageBytes("buyTicket"), nil
}

func (c *FSNContract) incAsset() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber

	assetValueChangeParamEx, err := c.GetAssetValueChangeExParam(from)
	if err != nil {
		return nil, err
	}
	if err := assetValueChangeParamEx.Check(height); err != nil {
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	assetID := assetValueChangeParamEx.AssetID
	asset, err := state.GetAsset(assetID)
	if err != nil {
		err = fmt.Errorf("asset not found")
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if !asset.CanChange {
		err = fmt.Errorf("asset can't inc")
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if asset.Owner != from {
		err = fmt.Errorf("can only be changed by owner")
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	state.AddBalance(assetValueChangeParamEx.To, assetID, assetValueChangeParamEx.Value)
	asset.Total = asset.Total.Add(asset.Total, assetValueChangeParamEx.Value)
	err = state.UpdateAsset(asset)
	if err == nil {
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("AssetID", assetID))
	} else {
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", "error update asset"))
	}
	return OkToStorageBytes("incAsset"), nil
}

func (c *FSNContract) decAsset() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber

	assetValueChangeParamEx, err := c.GetAssetValueChangeExParam(from)
	if err != nil {
		return nil, err
	}
	if err := assetValueChangeParamEx.Check(height); err != nil {
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	assetID := assetValueChangeParamEx.AssetID
	asset, err := state.GetAsset(assetID)
	if err != nil {
		err = fmt.Errorf("asset not found")
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if !asset.CanChange {
		err = fmt.Errorf("asset can't dec")
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if asset.Owner != from {
		err = fmt.Errorf("can only be changed by owner")
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if state.GetBalance(assetID, assetValueChangeParamEx.To).Cmp(assetValueChangeParamEx.Value) < 0 {
		err := fmt.Errorf("not enough asset")
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	state.SubBalance(assetValueChangeParamEx.To, assetID, assetValueChangeParamEx.Value)
	asset.Total = asset.Total.Sub(asset.Total, assetValueChangeParamEx.Value)
	err = state.UpdateAsset(asset)
	if err == nil {
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("AssetID", assetID))
	} else {
		c.addLog(common.AssetValueChangeFunc, *assetValueChangeParamEx, common.NewKeyValue("Error", "error update asset"))
	}
	return OkToStorageBytes("decAsset"), nil
}

func (c *FSNContract) sendAsset() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber

	sendAssetParam, err := c.GetSendAssetParam(state)
	if err != nil {
		return nil, err
	}
	if err := sendAssetParam.Check(height); err != nil {
		c.addLog(common.SendAssetFunc, *sendAssetParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	assetID := sendAssetParam.AssetID
	if state.GetBalance(assetID, from).Cmp(sendAssetParam.Value) < 0 {
		err := fmt.Errorf("not enough asset")
		c.addLog(common.SendAssetFunc, *sendAssetParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	state.SubBalance(from, assetID, sendAssetParam.Value)
	state.AddBalance(sendAssetParam.To, assetID, sendAssetParam.Value)
	c.addLog(common.SendAssetFunc, *sendAssetParam, common.NewKeyValue("AssetID", assetID))
	return OkToStorageBytes("sendAsset"), nil
}

func (c *FSNContract) genAsset() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber

	genAssetParam, err := c.GetGenAssetParam()
	if err != nil {
		return nil, err
	}
	if err := genAssetParam.Check(height); err != nil {
		c.addLog(common.GenAssetFunc, *genAssetParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	asset := genAssetParam.ToAsset()
	asset.ID = c.getIDHash()
	asset.Owner = from
	if err := state.GenAsset(asset); err != nil {
		c.addLog(common.GenAssetFunc, *genAssetParam, common.NewKeyValue("Error", "unable to gen asset"))
		return nil, err
	}
	state.AddBalance(from, asset.ID, asset.Total)
	c.addLog(common.GenAssetFunc, *genAssetParam, common.NewKeyValue("AssetID", asset.ID))
	return OkToStorageBytes("genAsset"), nil
}

func (c *FSNContract) genNotation() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()

	if err := state.GenNotation(from); err != nil {
		c.addLog(common.GenNotationFunc, "", common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	notation := state.GetNotation(from)
	c.addLog(common.GenNotationFunc, "", common.NewKeyValue("notation", notation))
	return OkToStorageBytes("genNotation"), nil
}

func (c *FSNContract) makeSwap() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	notation := state.GetNotation(from)
	makeSwapParam, err := c.GetMakeSwapParam()
	if err != nil {
		return nil, err
	}

	swapId := c.getIDHash()

	_, err = state.GetSwap(swapId)
	if err == nil {
		err = fmt.Errorf("Swap already exist")
		c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := makeSwapParam.Check(height, timestamp); err != nil {
		c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	var useAsset bool
	var total *big.Int
	var needValue *common.TimeLock

	fromAssetID := makeSwapParam.FromAssetID
	toAssetID := makeSwapParam.ToAssetID

	if _, err := state.GetAsset(toAssetID); err != nil {
		err := fmt.Errorf("ToAssetID's asset not found")
		c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if fromAssetID == common.OwnerUSANAssetID {
		if notation == 0 {
			err := fmt.Errorf("the from address does not have a notation")
			c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}
		makeSwapParam.MinFromAmount = big.NewInt(1)
		makeSwapParam.SwapSize = big.NewInt(1)
		makeSwapParam.FromStartTime = common.TimeLockNow
		makeSwapParam.FromEndTime = common.TimeLockForever
		useAsset = true
		total = new(big.Int).Mul(makeSwapParam.MinFromAmount, makeSwapParam.SwapSize)
	} else {
		total = new(big.Int).Mul(makeSwapParam.MinFromAmount, makeSwapParam.SwapSize)
		start := makeSwapParam.FromStartTime
		end := makeSwapParam.FromEndTime
		useAsset = start == common.TimeLockNow && end == common.TimeLockForever
		if useAsset == false {
			needValue = common.NewTimeLock(&common.TimeLockItem{
				StartTime: common.MaxUint64(start, timestamp),
				EndTime:   end,
				Value:     total,
			})
			if err := needValue.IsValid(); err != nil {
				c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
				return nil, err
			}
		}
	}
	swap := common.Swap{
		ID:            swapId,
		Owner:         from,
		FromAssetID:   fromAssetID,
		FromStartTime: makeSwapParam.FromStartTime,
		FromEndTime:   makeSwapParam.FromEndTime,
		MinFromAmount: makeSwapParam.MinFromAmount,
		ToAssetID:     toAssetID,
		ToStartTime:   makeSwapParam.ToStartTime,
		ToEndTime:     makeSwapParam.ToEndTime,
		MinToAmount:   makeSwapParam.MinToAmount,
		SwapSize:      makeSwapParam.SwapSize,
		Targes:        makeSwapParam.Targes,
		Time:          new(big.Int).SetUint64(timestamp),
		Description:   makeSwapParam.Description,
		Notation:      notation,
	}

	if fromAssetID == common.OwnerUSANAssetID {
		if err := state.AddSwap(swap); err != nil {
			c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", "can't add swap"))
			return nil, err
		}
	} else {
		if useAsset == true {
			if state.GetBalance(fromAssetID, from).Cmp(total) < 0 {
				err = fmt.Errorf("not enough from asset")
				c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
				return nil, err
			}
		} else {
			available := state.GetTimeLockBalance(fromAssetID, from)
			if available.Cmp(needValue) < 0 {
				if state.GetBalance(fromAssetID, from).Cmp(total) < 0 {
					err = fmt.Errorf("not enough time lock or asset balance")
					c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
					return nil, err
				}

				// subtract the asset from the balance
				state.SubBalance(from, fromAssetID, total)

				totalValue := common.NewTimeLock(&common.TimeLockItem{
					StartTime: timestamp,
					EndTime:   common.TimeLockForever,
					Value:     total,
				})
				state.AddTimeLockBalance(from, fromAssetID, totalValue, height, timestamp)

			}
		}

		if err := state.AddSwap(swap); err != nil {
			err = fmt.Errorf("can't add swap")
			c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}

		// take from the owner the asset
		if useAsset == true {
			state.SubBalance(from, fromAssetID, total)
		} else {
			state.SubTimeLockBalance(from, fromAssetID, needValue, height, timestamp)
		}
	}
	c.addLog(common.MakeSwapFunc, *makeSwapParam, common.NewKeyValue("SwapID", swap.ID))
	return OkToStorageBytes("makeSwap"), nil
}

func (c *FSNContract) recallSwap() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	recallSwapParam, err := c.GetRecallSwapParam()
	if err != nil {
		return nil, err
	}

	swapID := recallSwapParam.SwapID
	swap, err := state.GetSwap(swapID)
	if err != nil {
		err = fmt.Errorf("Swap not found")
		c.addLog(common.RecallSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if swap.Owner != from {
		err = fmt.Errorf("Must be swap onwer can recall")
		c.addLog(common.RecallSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := recallSwapParam.Check(height, &swap); err != nil {
		c.addLog(common.RecallSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := state.RemoveSwap(swap.ID); err != nil {
		err = fmt.Errorf("Unable to remove swap")
		c.addLog(common.RecallSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if swap.FromAssetID != common.OwnerUSANAssetID {
		total := new(big.Int).Mul(swap.MinFromAmount, swap.SwapSize)
		start := swap.FromStartTime
		end := swap.FromEndTime
		useAsset := start == common.TimeLockNow && end == common.TimeLockForever

		// return to the owner the balance
		if useAsset == true {
			state.AddBalance(from, swap.FromAssetID, total)
		} else {
			needValue := common.NewTimeLock(&common.TimeLockItem{
				StartTime: common.MaxUint64(start, timestamp),
				EndTime:   end,
				Value:     total,
			})
			if err := needValue.IsValid(); err == nil {
				state.AddTimeLockBalance(from, swap.FromAssetID, needValue, height, timestamp)
			}
		}
	}
	c.addLog(common.RecallSwapFunc, *recallSwapParam, common.NewKeyValue("SwapID", swap.ID))
	return OkToStorageBytes("recallSwap"), nil
}

func (c *FSNContract) takeSwap() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	takeSwapParam, err := c.GetTakeSwapParam()
	if err != nil {
		return nil, err
	}

	swapID := takeSwapParam.SwapID
	swap, err := state.GetSwap(swapID)
	if err != nil {
		err = fmt.Errorf("Swap not found")
		c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := takeSwapParam.Check(height, &swap, timestamp); err != nil {
		c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := common.CheckSwapTargets(swap.Targes, from); err != nil {
		c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	var usanSwap bool
	if swap.FromAssetID == common.OwnerUSANAssetID {
		notation := state.GetNotation(swap.Owner)
		if notation == 0 || notation != swap.Notation {
			err := fmt.Errorf("notation in swap is no longer valid")
			c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}
		usanSwap = true
	} else {
		usanSwap = false
	}

	fromTotal := new(big.Int).Mul(swap.MinFromAmount, takeSwapParam.Size)
	fromStart := swap.FromStartTime
	fromEnd := swap.FromEndTime
	fromUseAsset := fromStart == common.TimeLockNow && fromEnd == common.TimeLockForever

	toTotal := new(big.Int).Mul(swap.MinToAmount, takeSwapParam.Size)
	toStart := swap.ToStartTime
	toEnd := swap.ToEndTime
	toUseAsset := toStart == common.TimeLockNow && toEnd == common.TimeLockForever

	var fromNeedValue *common.TimeLock
	var toNeedValue *common.TimeLock

	if fromUseAsset == false {
		fromNeedValue = common.NewTimeLock(&common.TimeLockItem{
			StartTime: common.MaxUint64(fromStart, timestamp),
			EndTime:   fromEnd,
			Value:     fromTotal,
		})
	}
	if toUseAsset == false {
		toNeedValue = common.NewTimeLock(&common.TimeLockItem{
			StartTime: common.MaxUint64(toStart, timestamp),
			EndTime:   toEnd,
			Value:     toTotal,
		})
	}

	if toUseAsset == true {
		if state.GetBalance(swap.ToAssetID, from).Cmp(toTotal) < 0 {
			err = fmt.Errorf("not enough from asset")
			c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}
	} else {
		isValid := true
		if err := toNeedValue.IsValid(); err != nil {
			isValid = false
		}
		available := state.GetTimeLockBalance(swap.ToAssetID, from)
		if isValid && available.Cmp(toNeedValue) < 0 {
			if state.GetBalance(swap.ToAssetID, from).Cmp(toTotal) < 0 {
				err = fmt.Errorf("not enough time lock or asset balance")
				c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
				return nil, err
			}

			// subtract the asset from the balance
			state.SubBalance(from, swap.ToAssetID, toTotal)

			totalValue := common.NewTimeLock(&common.TimeLockItem{
				StartTime: timestamp,
				EndTime:   common.TimeLockForever,
				Value:     toTotal,
			})
			state.AddTimeLockBalance(from, swap.ToAssetID, totalValue, height, timestamp)

		}
	}

	swapDeleted := "false"

	if swap.SwapSize.Cmp(takeSwapParam.Size) == 0 {
		if err := state.RemoveSwap(swap.ID); err != nil {
			c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", "can't remove swap"))
			return nil, err
		}
		swapDeleted = "true"
	} else {
		swap.SwapSize = swap.SwapSize.Sub(swap.SwapSize, takeSwapParam.Size)
		if err := state.UpdateSwap(swap); err != nil {
			c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", "can't update swap"))
			return nil, err
		}
	}

	if toUseAsset == true {
		state.AddBalance(swap.Owner, swap.ToAssetID, toTotal)
		state.SubBalance(from, swap.ToAssetID, toTotal)
	} else {
		if err := toNeedValue.IsValid(); err == nil {
			state.AddTimeLockBalance(swap.Owner, swap.ToAssetID, toNeedValue, height, timestamp)
			state.SubTimeLockBalance(from, swap.ToAssetID, toNeedValue, height, timestamp)
		}
	}

	// credit the taker
	if usanSwap {
		err := state.TransferNotation(swap.Notation, swap.Owner, from)
		if err != nil {
			c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}
	} else {
		if fromUseAsset == true {
			state.AddBalance(from, swap.FromAssetID, fromTotal)
		} else {
			if err := fromNeedValue.IsValid(); err == nil {
				state.AddTimeLockBalance(from, swap.FromAssetID, fromNeedValue, height, timestamp)
			}
		}
	}
	c.addLog(common.TakeSwapFunc, *takeSwapParam, common.NewKeyValue("SwapID", swap.ID), common.NewKeyValue("Deleted", swapDeleted))
	return OkToStorageBytes("takeSwap"), nil
}

func (c *FSNContract) makeMultiSwap() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	makeSwapParam, err := c.GetMakeMultiSwapParam()
	if err != nil {
		return nil, err
	}

	notation := state.GetNotation(from)
	swapID := c.getIDHash()

	_, err = state.GetSwap(swapID)
	if err == nil {
		err = fmt.Errorf("Swap already exist")
		c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := makeSwapParam.Check(height, timestamp); err != nil {
		c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	fromAssetIDs := makeSwapParam.FromAssetID
	toAssetIDs := makeSwapParam.ToAssetID

	for _, toAssetID := range toAssetIDs {
		if _, err := state.GetAsset(toAssetID); err != nil {
			err := fmt.Errorf("ToAssetID's asset not found")
			c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}
	}

	ln := len(fromAssetIDs)

	useAsset := make([]bool, ln)
	total := make([]*big.Int, ln)
	needValue := make([]*common.TimeLock, ln)

	accountBalances := make(map[common.Hash]*big.Int)
	accountTimeLockBalances := make(map[common.Hash]*common.TimeLock)

	totalInterest := big.NewInt(0)
	isMortgage := common.IsMortgageEnabled(height) && makeSwapParam.IsMortgage()
	if isMortgage {
		totalInterest = makeSwapParam.GetInterest()
		accountBalances[common.SystemAssetID] = state.GetBalance(common.SystemAssetID, from)
	}

	for i := 0; i < ln; i++ {
		if _, exist := accountBalances[fromAssetIDs[i]]; !exist {
			balance := state.GetBalance(fromAssetIDs[i], from)
			timelock := state.GetTimeLockBalance(fromAssetIDs[i], from)
			accountBalances[fromAssetIDs[i]] = new(big.Int).Set(balance)
			accountTimeLockBalances[fromAssetIDs[i]] = timelock.Clone()
		}

		total[i] = new(big.Int).Mul(makeSwapParam.MinFromAmount[i], makeSwapParam.SwapSize)
		start := makeSwapParam.FromStartTime[i]
		end := makeSwapParam.FromEndTime[i]
		useAsset[i] = start == common.TimeLockNow && end == common.TimeLockForever
		if useAsset[i] == false {
			needValue[i] = common.NewTimeLock(&common.TimeLockItem{
				StartTime: common.MaxUint64(start, timestamp),
				EndTime:   end,
				Value:     total[i],
			})
			if err := needValue[i].IsValid(); err != nil {
				c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
				return nil, err
			}
		}

	}
	swap := common.MultiSwap{
		ID:            swapID,
		Owner:         from,
		FromAssetID:   fromAssetIDs,
		FromStartTime: makeSwapParam.FromStartTime,
		FromEndTime:   makeSwapParam.FromEndTime,
		MinFromAmount: makeSwapParam.MinFromAmount,
		ToAssetID:     toAssetIDs,
		ToStartTime:   makeSwapParam.ToStartTime,
		ToEndTime:     makeSwapParam.ToEndTime,
		MinToAmount:   makeSwapParam.MinToAmount,
		SwapSize:      makeSwapParam.SwapSize,
		Targes:        makeSwapParam.Targes,
		Time:          new(big.Int).SetUint64(timestamp),
		Description:   makeSwapParam.Description,
		Notation:      notation,
	}

	// check balances first
	for i := 0; i < ln; i++ {
		balance := accountBalances[fromAssetIDs[i]]
		timeLockBalance := accountTimeLockBalances[fromAssetIDs[i]]
		if useAsset[i] == true {
			if balance.Cmp(total[i]) < 0 {
				err = fmt.Errorf("not enough from asset")
				c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
				return nil, err
			}
			balance.Sub(balance, total[i])
		} else {
			if timeLockBalance.Cmp(needValue[i]) < 0 {
				if balance.Cmp(total[i]) < 0 {
					err = fmt.Errorf("not enough time lock or asset balance")
					c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
					return nil, err
				}

				balance.Sub(balance, total[i])
				totalValue := common.NewTimeLock(&common.TimeLockItem{
					StartTime: timestamp,
					EndTime:   common.TimeLockForever,
					Value:     total[i],
				})
				timeLockBalance.Add(timeLockBalance, totalValue)
			}
			timeLockBalance.Sub(timeLockBalance, needValue[i])
		}
	}
	if totalInterest.Sign() > 0 && accountBalances[common.SystemAssetID].Cmp(totalInterest) < 0 {
		err = fmt.Errorf("not enough interest asset")
		c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	// then deduct
	var deductErr error
	for i := 0; i < ln; i++ {
		if useAsset[i] == true {
			if state.GetBalance(fromAssetIDs[i], from).Cmp(total[i]) < 0 {
				deductErr = fmt.Errorf("not enough from asset")
				break
			}
		} else {
			available := state.GetTimeLockBalance(fromAssetIDs[i], from)
			if available.Cmp(needValue[i]) < 0 {

				if state.GetBalance(fromAssetIDs[i], from).Cmp(total[i]) < 0 {
					deductErr = fmt.Errorf("not enough time lock or asset balance")
					break
				}

				// subtract the asset from the balance
				state.SubBalance(from, fromAssetIDs[i], total[i])

				totalValue := common.NewTimeLock(&common.TimeLockItem{
					StartTime: timestamp,
					EndTime:   common.TimeLockForever,
					Value:     total[i],
				})
				state.AddTimeLockBalance(from, fromAssetIDs[i], totalValue, height, timestamp)
			}
		}

		// take from the owner the asset
		if useAsset[i] == true {
			state.SubBalance(from, fromAssetIDs[i], total[i])
		} else {
			state.SubTimeLockBalance(from, fromAssetIDs[i], needValue[i], height, timestamp)
		}
	}

	if deductErr == nil && totalInterest.Sign() > 0 {
		// take from the owner the interest
		if state.GetBalance(common.SystemAssetID, from).Cmp(totalInterest) < 0 {
			deductErr = fmt.Errorf("not enough interest asset")
		} else {
			state.SubBalance(from, common.SystemAssetID, totalInterest)
		}
	}

	if deductErr != nil {
		c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", deductErr.Error()))
		return nil, deductErr
	}

	if err := state.AddMultiSwap(swap); err != nil {
		c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	c.addLog(common.MakeMultiSwapFunc, *makeSwapParam, common.NewKeyValue("SwapID", swap.ID))
	return OkToStorageBytes("makeMultiSwap"), nil
}

func (c *FSNContract) recallMultiSwap() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	recallSwapParam, err := c.GetRecallMultiSwapParam()
	if err != nil {
		return nil, err
	}

	swapID := recallSwapParam.SwapID
	swap, err := state.GetMultiSwap(swapID)
	if err != nil {
		err = fmt.Errorf("Swap not found")
		c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := recallSwapParam.Check(height, &swap); err != nil {
		c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	isMortgage := common.IsMortgageEnabled(height) && swap.IsMortgage()
	isDealedMortgage := isMortgage && swap.SwapSize.Sign() == 0

	returnFromAssetTo := func(receiver common.Address) {
		swapSize := swap.SwapSize
		if isDealedMortgage {
			swapSize = big.NewInt(1)
		}

		for i := 0; i < len(swap.FromAssetID); i++ {
			total := new(big.Int).Mul(swap.MinFromAmount[i], swapSize)
			start := swap.FromStartTime[i]
			end := swap.FromEndTime[i]
			useAsset := start == common.TimeLockNow && end == common.TimeLockForever

			// return to the owner the balance
			if useAsset == true {
				state.AddBalance(receiver, swap.FromAssetID[i], total)
			} else {
				needValue := common.NewTimeLock(&common.TimeLockItem{
					StartTime: common.MaxUint64(start, timestamp),
					EndTime:   end,
					Value:     total,
				})

				if err := needValue.IsValid(); err == nil {
					state.AddTimeLockBalance(receiver, swap.FromAssetID[i], needValue, height, timestamp)
				}
			}
		}
	}

	returnInterestTo := func(receiver common.Address) {
		if isMortgage {
			totalInterest := swap.GetInterest()
			if totalInterest.Sign() > 0 {
				state.AddBalance(receiver, common.SystemAssetID, totalInterest)
			}
		}
	}

	if !isDealedMortgage {
		if swap.Owner != from {
			err = fmt.Errorf("Must be swap onwer can recall")
			c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}

		if err := state.RemoveMultiSwap(swap.ID); err != nil {
			err = fmt.Errorf("Unable to remove swap")
			c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}

		// return to the owner the balance
		returnFromAssetTo(from)

		// return to the owner the interest
		returnInterestTo(from)

		c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("SwapID", swap.ID))
		return OkToStorageBytes("recallMultiSwap"), nil // end of non-dealed
	}

	// process mortgage (has been taken by a lender)
	lender := swap.GetLender()
	lastEnd := swap.GetLastToEndTime()
	isExpired := timestamp > lastEnd

	if isExpired {
		if from != lender {
			err = fmt.Errorf("the mortgage is expired, only the lender can process it")
			c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}
		if err := state.RemoveMultiSwap(swap.ID); err != nil {
			err = fmt.Errorf("Unable to remove swap")
			c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
			return nil, err
		}

		// give to the lender the mortgage
		returnFromAssetTo(lender)

		// give to the lender the interest
		returnInterestTo(lender)

		c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("SwapID", swap.ID), common.NewKeyValue("Operator", "Lender"))
		return OkToStorageBytes("recallMultiSwap"), nil // end of expired
	}

	if from != swap.Owner {
		err = fmt.Errorf("the mortgage is still effective, only the borrower can process it")
		c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	// give the lender the borrowed balance
	if err := c.transferToAssets(from, lender, &swap, big.NewInt(1)); err != nil {
		c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := state.RemoveMultiSwap(swap.ID); err != nil {
		err = fmt.Errorf("Unable to remove swap")
		c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	// return to the owner the balance
	returnFromAssetTo(from)

	// give to the lender the interest
	returnInterestTo(lender)

	c.addLog(common.RecallMultiSwapFunc, *recallSwapParam, common.NewKeyValue("SwapID", swap.ID), common.NewKeyValue("Operator", "Borrower"))
	return OkToStorageBytes("recallMultiSwap"), nil
}

func (c *FSNContract) takeMultiSwap() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	takeSwapParam, err := c.GetTakeMultiSwapParam()
	if err != nil {
		return nil, err
	}

	swapID := takeSwapParam.SwapID
	swap, err := state.GetMultiSwap(swapID)
	if err != nil {
		err = fmt.Errorf("Swap not found")
		c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := takeSwapParam.Check(height, &swap, timestamp); err != nil {
		c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	if err := common.CheckSwapTargets(swap.Targes, from); err != nil {
		c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	lnFrom := len(swap.FromAssetID)

	fromUseAsset := make([]bool, lnFrom)
	fromTotal := make([]*big.Int, lnFrom)
	fromStart := make([]uint64, lnFrom)
	fromEnd := make([]uint64, lnFrom)
	fromNeedValue := make([]*common.TimeLock, lnFrom)
	for i := 0; i < lnFrom; i++ {
		fromTotal[i] = new(big.Int).Mul(swap.MinFromAmount[i], takeSwapParam.Size)
		fromStart[i] = swap.FromStartTime[i]
		fromEnd[i] = swap.FromEndTime[i]
		fromUseAsset[i] = fromStart[i] == common.TimeLockNow && fromEnd[i] == common.TimeLockForever

		if fromUseAsset[i] == false {
			fromNeedValue[i] = common.NewTimeLock(&common.TimeLockItem{
				StartTime: common.MaxUint64(fromStart[i], timestamp),
				EndTime:   fromEnd[i],
				Value:     fromTotal[i],
			})
		}
	}

	if err := c.transferToAssets(from, swap.Owner, &swap, takeSwapParam.Size); err != nil {
		c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	swapDeleted := "false"

	isMortgage := common.IsMortgageEnabled(height) && swap.IsMortgage()

	if isMortgage {
		swap.SwapSize = swap.SwapSize.Sub(swap.SwapSize, takeSwapParam.Size)
		swap.Targes = append(swap.Targes, from) // record the lender
		if err := state.UpdateMultiSwap(swap); err != nil {
			c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("Error", "can't update swap"))
			return nil, err
		}
	} else if swap.SwapSize.Cmp(takeSwapParam.Size) == 0 {
		if err := state.RemoveMultiSwap(swap.ID); err != nil {
			c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("Error", "can't remove swap"))
			return nil, err
		}
		swapDeleted = "true"
	} else {
		swap.SwapSize = swap.SwapSize.Sub(swap.SwapSize, takeSwapParam.Size)
		if err := state.UpdateMultiSwap(swap); err != nil {
			c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("Error", "can't update swap"))
			return nil, err
		}
	}

	// credit the swap take with the from assets
	for i := 0; i < lnFrom; i++ {
		if isMortgage {
			break
		}
		if fromUseAsset[i] == true {
			state.AddBalance(from, swap.FromAssetID[i], fromTotal[i])
		} else {
			if err := fromNeedValue[i].IsValid(); err == nil {
				state.AddTimeLockBalance(from, swap.FromAssetID[i], fromNeedValue[i], height, timestamp)
			}
		}
	}
	c.addLog(common.TakeMultiSwapFunc, *takeSwapParam, common.NewKeyValue("SwapID", swap.ID), common.NewKeyValue("Deleted", swapDeleted))
	return OkToStorageBytes("takeMultiSwap"), nil
}

func (c *FSNContract) transferToAssets(from, to common.Address, swap *common.MultiSwap, size *big.Int) error {
	state := c.evm.StateDB
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	lnTo := len(swap.ToAssetID)

	toUseAsset := make([]bool, lnTo)
	toTotal := make([]*big.Int, lnTo)
	toStart := make([]uint64, lnTo)
	toEnd := make([]uint64, lnTo)
	toNeedValue := make([]*common.TimeLock, lnTo)

	accountBalances := make(map[common.Hash]*big.Int)
	accountTimeLockBalances := make(map[common.Hash]*common.TimeLock)

	for i := 0; i < lnTo; i++ {
		if _, exist := accountBalances[swap.ToAssetID[i]]; !exist {
			balance := state.GetBalance(swap.ToAssetID[i], from)
			timelock := state.GetTimeLockBalance(swap.ToAssetID[i], from)
			accountBalances[swap.ToAssetID[i]] = new(big.Int).Set(balance)
			accountTimeLockBalances[swap.ToAssetID[i]] = timelock.Clone()
		}

		toTotal[i] = new(big.Int).Mul(swap.MinToAmount[i], size)
		toStart[i] = swap.ToStartTime[i]
		toEnd[i] = swap.ToEndTime[i]
		toUseAsset[i] = toStart[i] == common.TimeLockNow && toEnd[i] == common.TimeLockForever
		if toUseAsset[i] == false {
			toNeedValue[i] = common.NewTimeLock(&common.TimeLockItem{
				StartTime: common.MaxUint64(toStart[i], timestamp),
				EndTime:   toEnd[i],
				Value:     toTotal[i],
			})
		}
	}

	// check to account balances
	for i := 0; i < lnTo; i++ {
		balance := accountBalances[swap.ToAssetID[i]]
		timeLockBalance := accountTimeLockBalances[swap.ToAssetID[i]]
		if toUseAsset[i] == true {
			if balance.Cmp(toTotal[i]) < 0 {
				return fmt.Errorf("not enough from asset")
			}
			balance.Sub(balance, toTotal[i])
		} else {
			if err := toNeedValue[i].IsValid(); err != nil {
				continue
			}
			if timeLockBalance.Cmp(toNeedValue[i]) < 0 {
				if balance.Cmp(toTotal[i]) < 0 {
					return fmt.Errorf("not enough time lock or asset balance")
				}

				balance.Sub(balance, toTotal[i])
				totalValue := common.NewTimeLock(&common.TimeLockItem{
					StartTime: timestamp,
					EndTime:   common.TimeLockForever,
					Value:     toTotal[i],
				})
				timeLockBalance.Add(timeLockBalance, totalValue)
			}
			timeLockBalance.Sub(timeLockBalance, toNeedValue[i])
		}
	}

	// then deduct
	var deductErr error
	for i := 0; i < lnTo; i++ {
		if toUseAsset[i] == true {
			if state.GetBalance(swap.ToAssetID[i], from).Cmp(toTotal[i]) < 0 {
				deductErr = fmt.Errorf("not enough from asset")
				break
			}
			state.SubBalance(from, swap.ToAssetID[i], toTotal[i])
		} else {
			if err := toNeedValue[i].IsValid(); err != nil {
				continue
			}
			available := state.GetTimeLockBalance(swap.ToAssetID[i], from)
			if available.Cmp(toNeedValue[i]) < 0 {

				if state.GetBalance(swap.ToAssetID[i], from).Cmp(toTotal[i]) < 0 {
					deductErr = fmt.Errorf("not enough time lock or asset balance")
					break
				}

				// subtract the asset from the balance
				state.SubBalance(from, swap.ToAssetID[i], toTotal[i])

				totalValue := common.NewTimeLock(&common.TimeLockItem{
					StartTime: timestamp,
					EndTime:   common.TimeLockForever,
					Value:     toTotal[i],
				})
				state.AddTimeLockBalance(from, swap.ToAssetID[i], totalValue, height, timestamp)
			}
			state.SubTimeLockBalance(from, swap.ToAssetID[i], toNeedValue[i], height, timestamp)
		}
	}

	if deductErr != nil {
		return deductErr
	}

	for i := 0; i < lnTo; i++ {
		if toUseAsset[i] == true {
			state.AddBalance(to, swap.ToAssetID[i], toTotal[i])
		} else {
			if err := toNeedValue[i].IsValid(); err == nil {
				state.AddTimeLockBalance(to, swap.ToAssetID[i], toNeedValue[i], height, timestamp)
			}
		}
	}

	return nil
}

func (c *FSNContract) ticketPrice() ([]byte, error) {
	height := common.BigMaxUint64
	ticketPrice := common.TicketPrice(height)
	return ticketPrice.Bytes(), nil
}

func (c *FSNContract) assetToTimeLock() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	timeLockParam, err := c.GetTimeLockParam(state)
	if err != nil {
		return nil, err
	}
	if err := timeLockParam.Check(height, timestamp); err != nil {
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	start := timeLockParam.StartTime
	end := timeLockParam.EndTime
	needValue := common.NewTimeLock(&common.TimeLockItem{
		StartTime: common.MaxUint64(start, timestamp),
		EndTime:   end,
		Value:     new(big.Int).SetBytes(timeLockParam.Value.Bytes()),
	})
	if err := needValue.IsValid(); err != nil {
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	assetID := timeLockParam.AssetID
	if state.GetBalance(assetID, from).Cmp(timeLockParam.Value) < 0 {
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("LockType", "AssetToTimeLock"), common.NewKeyValue("Error", "not enough asset"))
		return nil, fmt.Errorf("not enough asset")
	}
	state.SubBalance(from, assetID, timeLockParam.Value)

	totalValue := common.NewTimeLock(&common.TimeLockItem{
		StartTime: timestamp,
		EndTime:   common.TimeLockForever,
		Value:     new(big.Int).SetBytes(timeLockParam.Value.Bytes()),
	})
	if from == timeLockParam.To {
		state.AddTimeLockBalance(timeLockParam.To, assetID, totalValue, height, timestamp)
	} else {
		surplusValue := new(common.TimeLock).Sub(totalValue, needValue)
		if !surplusValue.IsEmpty() {
			state.AddTimeLockBalance(from, assetID, surplusValue, height, timestamp)
		}
		state.AddTimeLockBalance(timeLockParam.To, assetID, needValue, height, timestamp)
	}

	c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("LockType", "AssetToTimeLock"), common.NewKeyValue("AssetID", assetID))
	return OkToStorageBytes("assetToTimeLock"), nil
}

func (c *FSNContract) timeLockToAsset() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	timeLockParam, err := c.GetTimeLockParam(state)
	if err != nil {
		return nil, err
	}
	if err := timeLockParam.Check(height, timestamp); err != nil {
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	start := timeLockParam.StartTime
	end := timeLockParam.EndTime
	needValue := common.NewTimeLock(&common.TimeLockItem{
		StartTime: common.MaxUint64(start, timestamp),
		EndTime:   end,
		Value:     new(big.Int).SetBytes(timeLockParam.Value.Bytes()),
	})
	if err := needValue.IsValid(); err != nil {
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	assetID := timeLockParam.AssetID
	if state.GetTimeLockBalance(assetID, from).Cmp(needValue) < 0 {
		err := fmt.Errorf("not enough time lock balance")
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("LockType", "TimeLockToAsset"), common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	state.SubTimeLockBalance(from, assetID, needValue, height, timestamp)
	state.AddBalance(timeLockParam.To, assetID, timeLockParam.Value)
	c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("LockType", "TimeLockToAsset"), common.NewKeyValue("AssetID", assetID))
	return OkToStorageBytes("timeLockToAsset"), nil
}

func (c *FSNContract) timeLockToTimeLock() ([]byte, error) {
	state := c.evm.StateDB
	from := c.getFrom()
	height := c.evm.BlockNumber
	timestamp := c.getParentTime()

	timeLockParam, err := c.GetTimeLockParam(state)
	if err != nil {
		return nil, err
	}
	if err := timeLockParam.Check(height, timestamp); err != nil {
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	start := timeLockParam.StartTime
	end := timeLockParam.EndTime
	needValue := common.NewTimeLock(&common.TimeLockItem{
		StartTime: common.MaxUint64(start, timestamp),
		EndTime:   end,
		Value:     new(big.Int).SetBytes(timeLockParam.Value.Bytes()),
	})
	if err := needValue.IsValid(); err != nil {
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("Error", err.Error()))
		return nil, err
	}

	assetID := timeLockParam.AssetID
	if state.GetTimeLockBalance(assetID, from).Cmp(needValue) < 0 {
		err := fmt.Errorf("not enough time lock balance")
		c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("LockType", "TimeLockToTimeLock"), common.NewKeyValue("Error", err.Error()))
		return nil, err
	}
	state.SubTimeLockBalance(from, assetID, needValue, height, timestamp)
	state.AddTimeLockBalance(timeLockParam.To, assetID, needValue, height, timestamp)
	c.addLog(common.TimeLockFunc, *timeLockParam, common.NewKeyValue("LockType", "TimeLockToTimeLock"), common.NewKeyValue("AssetID", assetID))
	return OkToStorageBytes("timeLockToTimeLock"), nil
}

func (c *FSNContract) getAddressByNotation() ([]byte, error) {
	state := c.evm.StateDB
	notation := c.getUint64Param(1)
	address := (common.Address{})
	if addr, err := state.GetAddressByNotation(notation); err == nil {
		address = addr
	}
	return address[:], nil
}

func FSNContractGetAllBalances(state StateDB, data []byte) ([]byte, error) {
	address := common.BigToAddress(getBigInt(data, 4))
	balances := state.GetAllBalances(address)
	bs, _ := json.Marshal(balances)
	return ToStorageBytes(bs), nil
}

func (c *FSNContract) getAllBalances() ([]byte, error) {
	return FSNContractGetAllBalances(c.evm.StateDB, c.input)
}

func (c *FSNContract) getAllTimeLockBalances() ([]byte, error) {
	state := c.evm.StateDB
	address := c.getAddressParam(1)
	balances := state.GetAllTimeLockBalances(address)
	bs, _ := json.Marshal(balances)
	return ToStorageBytes(bs), nil
}

func (c *FSNContract) getAsset() ([]byte, error) {
	state := c.evm.StateDB
	assetID := c.getHashParam(1)
	asset, err := state.GetAsset(assetID)
	if err != nil {
		return nil, err
	}
	bs, _ := asset.MarshalJSON()
	return ToStorageBytes(bs), nil
}

func FSNContractGetBalance(state StateDB, data []byte) ([]byte, error) {
	assetID := common.BigToHash(getBigInt(data, 4))
	address := common.BigToAddress(getBigInt(data, 36))
	balance := state.GetBalance(assetID, address)
	return balance.Bytes(), nil
}

func (c *FSNContract) getBalance() ([]byte, error) {
	return FSNContractGetBalance(c.evm.StateDB, c.input)
}

func (c *FSNContract) getMultiSwap() ([]byte, error) {
	state := c.evm.StateDB
	swapID := c.getHashParam(1)
	swap, err := state.GetMultiSwap(swapID)
	if err != nil {
		return nil, err
	}
	bs, _ := json.Marshal(swap)
	return ToStorageBytes(bs[:]), nil
}

func (c *FSNContract) getNotation() ([]byte, error) {
	state := c.evm.StateDB
	address := c.getAddressParam(1)
	notation := state.GetNotation(address)
	return common.Uint64ToBytes(notation), nil
}

func (c *FSNContract) getSwap() ([]byte, error) {
	state := c.evm.StateDB
	swapID := c.getHashParam(1)
	swap, err := state.GetSwap(swapID)
	if err != nil {
		return nil, err
	}
	bs, _ := json.Marshal(swap)
	return ToStorageBytes(bs[:]), nil
}

func (c *FSNContract) getTimeLockBalance() ([]byte, error) {
	state := c.evm.StateDB
	assetID := c.getHashParam(1)
	address := c.getAddressParam(2)
	balance := state.GetTimeLockBalance(assetID, address)
	bs, _ := json.Marshal(balance)
	return ToStorageBytes(bs[:]), nil
}

func (c *FSNContract) getStakeInfo() ([]byte, error) {
	type Summary struct {
		TotalMiners  uint64 `json:"totalMiners"`
		TotalTickets uint64 `json:"totalTickets"`
	}
	type MineInfo struct {
		MineInfo map[common.Address]uint64 `json:"mineInfo"`
		Summary  Summary                   `json:"summary"`
	}
	state := c.evm.StateDB
	tickets, err := state.AllTickets()
	if err != nil {
		return nil, err
	}
	mineInfo := MineInfo{
		MineInfo: make(map[common.Address]uint64),
	}
	mineInfo.Summary.TotalTickets, mineInfo.Summary.TotalMiners = tickets.NumberOfTicketsAndOwners()
	for _, v := range tickets {
		mineInfo.MineInfo[v.Owner] = uint64(len(v.Tickets))
	}
	bs, _ := json.Marshal(mineInfo)
	return ToStorageBytes(bs[:]), nil
}

func (c *FSNContract) totalNumberOfTickets() ([]byte, error) {
	state := c.evm.StateDB
	numberOfTickets := uint64(0)
	if tickets, err := state.AllTickets(); err == nil {
		numberOfTickets = tickets.NumberOfTickets()
	}
	return common.Uint64ToBytes(numberOfTickets), nil
}

func (c *FSNContract) totalNumberOfTicketsByAddress() ([]byte, error) {
	state := c.evm.StateDB
	address := c.getAddressParam(1)
	numberOfTickets := uint64(0)
	if tickets, err := state.AllTickets(); err == nil {
		numberOfTickets = tickets.NumberOfTicketsByAddress(address)
	}
	return common.Uint64ToBytes(numberOfTickets), nil
}

func (c *FSNContract) getAllTickets() ([]byte, error) {
	state := c.evm.StateDB
	tickets, err := state.AllTickets()
	if err != nil {
		return nil, err
	}
	bs, _ := json.Marshal(tickets.ToMap())
	return ToStorageBytes(bs[:]), nil
}

func (c *FSNContract) getAllTicketsByAddress() ([]byte, error) {
	state := c.evm.StateDB
	address := c.getAddressParam(1)
	tickets, err := state.AllTickets()
	if err != nil {
		return nil, err
	}
	var bs []byte
	for _, v := range tickets {
		if v.Owner == address {
			bs, _ = json.Marshal(v.ToMap())
			break
		}
	}
	return ToStorageBytes(bs[:]), nil
}

func OkToStorageBytes(str string) []byte {
	return ToStorageBytes([]byte("OK: " + str))
}

func ErrToStorageBytes(err error) []byte {
	return ToStorageBytes([]byte("Error: " + err.Error()))
}

func ToStorageBytes(bs []byte) []byte {
	data := make([]byte, 0)
	data = append(data, common.LeftPadBytes([]byte{0x20}, 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(len(bs))).Bytes(), 32)...)
	data = append(data, bs...)
	return data
}

func (c *FSNContract) addLog(funcType common.FSNCallFunc, value interface{}, keyValues ...*common.KeyValue) {
	t := reflect.TypeOf(value)
	v := reflect.ValueOf(value)

	maps := make(map[string]interface{})
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			if v.Field(i).CanInterface() {
				maps[t.Field(i).Name] = v.Field(i).Interface()
			}
		}
	} else {
		maps["Base"] = value
	}

	for i := 0; i < len(keyValues); i++ {
		maps[keyValues[i].Key] = keyValues[i].Value
	}

	data, _ := json.Marshal(maps)

	topic := common.Hash{}
	topic[common.HashLength-1] = (uint8)(funcType)

	c.evm.StateDB.AddLog(&types.Log{
		Address:     common.FSNContractAddress,
		Topics:      []common.Hash{topic},
		Data:        data,
		BlockNumber: c.evm.BlockNumber.Uint64(),
	})
}
