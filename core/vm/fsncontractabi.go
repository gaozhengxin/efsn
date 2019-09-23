package vm

import (
	"regexp"
)

const FSNContractABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "totalNumberOfTickets",
		"outputs": [
			{
				"name": "",
				"type": "int32"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "swapid",
				"type": "bytes32"
			},
			{
				"name": "size",
				"type": "uint256"
			}
		],
		"name": "takeMultiSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "totalNumberOfTicketsByAddress",
		"outputs": [
			{
				"name": "",
				"type": "int32"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "ticketPrice",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "notation",
				"type": "uint64"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "USANsendAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "assetID",
				"type": "bytes32"
			}
		],
		"name": "getAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "fromAssetIDs",
				"type": "bytes32[]"
			},
			{
				"name": "fromAmounts",
				"type": "uint256[]"
			},
			{
				"name": "toAssetIDs",
				"type": "bytes32[]"
			},
			{
				"name": "toAmounts",
				"type": "uint256[]"
			},
			{
				"name": "swapSize",
				"type": "uint256"
			},
			{
				"name": "targets",
				"type": "address[]"
			},
			{
				"name": "description",
				"type": "string"
			}
		],
		"name": "makeMultiSwap2",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "swapID",
				"type": "bytes32"
			}
		],
		"name": "getSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "swapid",
				"type": "bytes32"
			},
			{
				"name": "size",
				"type": "uint256"
			}
		],
		"name": "takeSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "to",
				"type": "address"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "sendAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "start",
				"type": "uint64"
			},
			{
				"name": "end",
				"type": "uint64"
			}
		],
		"name": "buyTicket",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "fromAssetID",
				"type": "bytes32[]"
			},
			{
				"name": "fromStartTime",
				"type": "uint64[]"
			},
			{
				"name": "fromEndTime",
				"type": "uint64[]"
			},
			{
				"name": "minFromAmount",
				"type": "uint256[]"
			},
			{
				"name": "toAssetID",
				"type": "bytes32[]"
			},
			{
				"name": "toStartTime",
				"type": "uint64[]"
			},
			{
				"name": "toEndTime",
				"type": "uint64[]"
			},
			{
				"name": "minToAmount",
				"type": "uint256[]"
			},
			{
				"name": "swapSize",
				"type": "uint256"
			},
			{
				"name": "targets",
				"type": "address[]"
			},
			{
				"name": "description",
				"type": "string"
			}
		],
		"name": "makeMultiSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "assetID",
				"type": "bytes32"
			},
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "getBalance",
		"outputs": [
			{
				"name": "",
				"type": "uint256"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "to",
				"type": "address"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "incAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "decAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "swapid",
				"type": "bytes32"
			}
		],
		"name": "recallMultiSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "fromAssetID",
				"type": "bytes32"
			},
			{
				"name": "fromStartTime",
				"type": "uint64"
			},
			{
				"name": "fromEndTime",
				"type": "uint64"
			},
			{
				"name": "minFromAmount",
				"type": "uint256"
			},
			{
				"name": "toAssetID",
				"type": "bytes32"
			},
			{
				"name": "toStartTime",
				"type": "uint64"
			},
			{
				"name": "toEndTime",
				"type": "uint64"
			},
			{
				"name": "minToAmount",
				"type": "uint256"
			},
			{
				"name": "swapSize",
				"type": "uint256"
			},
			{
				"name": "targets",
				"type": "address[]"
			},
			{
				"name": "description",
				"type": "string"
			}
		],
		"name": "makeSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "notation",
				"type": "uint64"
			},
			{
				"name": "start",
				"type": "uint64"
			},
			{
				"name": "end",
				"type": "uint64"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "USANtimeLockToTimeLock",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "getStakeInfo",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "to",
				"type": "address"
			},
			{
				"name": "start",
				"type": "uint64"
			},
			{
				"name": "end",
				"type": "uint64"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "assetToTimeLock",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "genNotation",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "getAllTimeLockBalances",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "symbol",
				"type": "string"
			},
			{
				"name": "decimals",
				"type": "uint8"
			},
			{
				"name": "total",
				"type": "uint256"
			},
			{
				"name": "canChange",
				"type": "bool"
			},
			{
				"name": "description",
				"type": "string"
			}
		],
		"name": "genAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "notation",
				"type": "uint64"
			},
			{
				"name": "start",
				"type": "uint64"
			},
			{
				"name": "end",
				"type": "uint64"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "USANassetToTimeLock",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "allTicketsByAddress",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "notation",
				"type": "uint64"
			}
		],
		"name": "getAddressByNotation",
		"outputs": [
			{
				"name": "",
				"type": "address"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "getNotation",
		"outputs": [
			{
				"name": "",
				"type": "uint64"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "getAllBalances",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "to",
				"type": "address"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "timeLockToAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "swapid",
				"type": "bytes32"
			}
		],
		"name": "recallSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "assetID",
				"type": "bytes32"
			},
			{
				"name": "addr",
				"type": "address"
			}
		],
		"name": "getTimeLockBalance",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "allTickets",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "notation",
				"type": "uint64"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "USANtimeLockToAsset",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "asset",
				"type": "bytes32"
			},
			{
				"name": "to",
				"type": "address"
			},
			{
				"name": "start",
				"type": "uint64"
			},
			{
				"name": "end",
				"type": "uint64"
			},
			{
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "timeLockToTimeLock",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{
				"name": "swapID",
				"type": "bytes32"
			}
		],
		"name": "getMultiSwap",
		"outputs": [
			{
				"name": "",
				"type": "string"
			}
		],
		"payable": false,
		"stateMutability": "pure",
		"type": "function"
	}
]`

var FSNContractABICompact = regexp.MustCompile(`\s+`).ReplaceAllString(FSNContractABI, "")
