package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"
	w "github.com/FusionFoundation/efsn/cmd/fsn_mining_pool/withdraw"
	"github.com/FusionFoundation/efsn/crypto"
)

//curl -X POST -H "Content-Type":application/json --data '{"hash":"0x7dcab6207798cda324fd41b4c745f186a60b10834bc0c77a37e8808a13eea86f","address":"0x0122BF3930c1201A21133937Ad5C83Eb4dEd1b08","amount":"100","timestamp":"1572252192","sig":"f0ad9d40a5cb76ea1a89813290a8abfdce4046448f24a73342195c42945c19fe0dab909af713d1fddc5db99be37d21487e6f9422acfb16bfbdcb84fb6dcdd77101"}' http://0.0.0.0:9990

var (
	address string
	amount string
	pk string
)

func init() {
	flag.StringVar(&address, "a", "", "address hex string")
	flag.StringVar(&amount, "amount", "", "withdraw amount")
	flag.StringVar(&pk, "k", "", "private key hex string")
}

func main() {
	flag.Parse()
	pk = strings.TrimPrefix(pk, "0x")
	req := &w.WithdrawRequest{
		Address:address,
		Amount:amount,
		Timestamp:fmt.Sprintf("%v", time.Now().Unix()),
	}
	req.MakeHash()
	//fmt.Printf("req:\n%+v\n", req)
	priv, err := crypto.HexToECDSA(pk)
	if err != nil {
		panic(err)
	}
	err = w.SignWithdrawRequest(req, priv)
	if err != nil {
		panic(err)
	}
	if ok := req.VerifySignature(); ok {
		data, _ := json.Marshal(req)
		//fmt.Printf("data:\n%v\n", string(data))
		fmt.Printf(string(data))
	}
	return
}
