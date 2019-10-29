package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/FusionFoundation/efsn/cmd/fsn_mining_pool/withdraw"
	"github.com/FusionFoundation/efsn/log"
)

var port string

func ServerRun() {
	http.HandleFunc("/", Withdraw)
	if err := http.ListenAndServe("0.0.0.0:" + port, nil); err != nil {
		log.Error("Server failed", "error", err)
	}
}

func Withdraw(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))
	req := &withdraw.WithdrawRequest{}
	if err := json.Unmarshal(body, req); err != nil {
		fmt.Fprint(w, err.Error() + "\n")
		return
	}
	fmt.Printf("\n\nreq:\n%+v\n\n", req)
	if err := ValidateWithdraw(req); err != nil {
		fmt.Fprint(w, err.Error() + "\n")
		return
	}
	ret := <-WithdrawRetCh
	d, err := json.Marshal(ret)
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}
	fmt.Fprint(w, string(d))
	return
}
