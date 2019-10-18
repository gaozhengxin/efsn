package main

import (
	"encoding/json"
	"fmt"
	w "github.com/FusionFoundation/efsn/cmd/fsn_mining_pool/withdraw"
	"github.com/FusionFoundation/efsn/crypto"
)

//curl -X POST -H "Content-Type":application/json --data '{"hash":"0x85b69a0bbcc7c6d52338a9f78983405012443d4a1da9e3d9d808d472bd90db95","address":"0x0963a18ea497b7724340fdfe4ff6e060d3f9e388","amount":"100","sig":"b060ccafe5f0dbe1e423c1d6fac06d9a21f25f5b7666084e741b4c39e5db485a109f52dc4570afd5926e12c0605e75eb331584f95ac4646ad7f1d3a8e12b9c1e00"}' http://0.0.0.0:9990

func main() {
	req := &w.WithdrawRequest{
		Address:"0x0963a18ea497b7724340fdfe4ff6e060d3f9e388",
		Amount:"100",
	}
	req.MakeHash()
	fmt.Printf("req:\n%+v\n", req)
	priv, _ := crypto.HexToECDSA("40d6e64ce085269869b178c23a786e499ff2d6a5334fe45964211d25bea973bf")
	err := w.SignWithdrawRequest(req, priv)
	if err != nil {
		panic(err)
	}
	if ok := req.VerifySignature(); ok {
		data, _ := json.Marshal(req)
		fmt.Printf("data:\n%v\n", string(data))
	}
	return
}
