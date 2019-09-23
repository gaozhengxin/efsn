package common

import (
	"fmt"

	"github.com/FusionFoundation/efsn/common/hexutil"
)

// Contract FuncHashes
const (
	USANsendAssetFH                 = "192733f2"
	USANassetToTimeLockFH           = "a230d908"
	USANtimeLockToTimeLockFH        = "69309593"
	USANtimeLockToAssetFH           = "e6dde89b"
	AllTicketsFH                    = "d83113d5"
	AllTicketsByAddressFH           = "a4945860"
	AssetToTimeLockFH               = "7654344f"
	BuyTicketFH                     = "4768c13c"
	DecAssetFH                      = "5712063a"
	GenAssetFH                      = "9c261a0e"
	GenNotationFH                   = "782fe0ea"
	GetAddressByNotationFH          = "a78c7c2d"
	GetAllBalancesFH                = "c53d6ce1"
	GetAllTimeLockBalancesFH        = "8cc32418"
	GetAssetFH                      = "2cc3ce80"
	GetBalanceFH                    = "530e931c"
	GetMultiSwapFH                  = "fa0e4367"
	GetNotationFH                   = "bf0e2d63"
	GetStakeInfoFH                  = "74a4030f"
	GetSwapFH                       = "3da0e66e"
	GetTimeLockBalanceFH            = "ca53c58b"
	IncAssetFH                      = "555ba438"
	MakeMultiSwapFH                 = "4a7f8c97"
	MakeMultiSwap2FH                = "2f4f9edc"
	MakeSwapFH                      = "68ee27ba"
	RecallMultiSwapFH               = "5870a429"
	RecallSwapFH                    = "c72b372f"
	SendAssetFH                     = "404166b6"
	TakeMultiSwapFH                 = "0bc9ac2f"
	TakeSwapFH                      = "3e53f03e"
	TicketPriceFH                   = "1209b1f6"
	TimeLockToAssetFH               = "c62f2af9"
	TimeLockToTimeLockFH            = "e94aa87d"
	TotalNumberOfTicketsFH          = "00746702"
	TotalNumberOfTicketsByAddressFH = "1012275f"
)

func GetFsnFuncHash(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	return hexutil.Encode(data[:4])[2:]
}

func GetFsnFuncType(input []byte) FSNCallFunc {
	switch GetFsnFuncHash(input) {
	case SendAssetFH, USANsendAssetFH:
		return SendAssetFunc
	case AssetToTimeLockFH, USANassetToTimeLockFH,
		TimeLockToAssetFH, USANtimeLockToAssetFH,
		TimeLockToTimeLockFH, USANtimeLockToTimeLockFH:
		return TimeLockFunc
	case BuyTicketFH:
		return BuyTicketFunc
	case DecAssetFH, IncAssetFH:
		return AssetValueChangeFunc
	case GenAssetFH:
		return GenAssetFunc
	case GenNotationFH:
		return GenNotationFunc
	case MakeMultiSwapFH, MakeMultiSwap2FH:
		return MakeMultiSwapFunc
	case MakeSwapFH:
		return MakeSwapFuncExt
	case RecallMultiSwapFH:
		return RecallMultiSwapFunc
	case RecallSwapFH:
		return RecallSwapFunc
	case TakeMultiSwapFH:
		return TakeMultiSwapFunc
	case TakeSwapFH:
		return TakeSwapFuncExt
	}
	return EmptyFunc
}

func IsUsingUSAN(input []byte) bool {
	switch GetFsnFuncHash(input) {
	case USANsendAssetFH,
		USANassetToTimeLockFH,
		USANtimeLockToTimeLockFH,
		USANtimeLockToAssetFH:
		return true
	}
	return false
}

func IsMakeMultiSwap2(input []byte) bool {
	return GetFsnFuncHash(input) == MakeMultiSwap2FH
}

func IsIncAsset(input []byte) bool {
	return GetFsnFuncHash(input) == IncAssetFH
}

func GetTimeLockType(input []byte) (TimeLockType, error) {
	switch GetFsnFuncHash(input) {
	case AssetToTimeLockFH, USANassetToTimeLockFH:
		return AssetToTimeLock, nil
	case TimeLockToAssetFH, USANtimeLockToAssetFH:
		return TimeLockToAsset, nil
	case TimeLockToTimeLockFH, USANtimeLockToTimeLockFH:
		return TimeLockToTimeLock, nil
	}
	return AssetToTimeLock, fmt.Errorf("unknown timelock type")
}
