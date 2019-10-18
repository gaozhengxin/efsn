package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"
	"github.com/FusionFoundation/efsn/common"
	"github.com/FusionFoundation/efsn/crypto"
)

type WithdrawRequest struct {
	Hash    string `json:"hash,omitempty"`
	Address string `json:"address"`
	Amount  string `json:"amount"`
	//Timestamp uint64 `json:"timestamp"` ???
	Sig     string `json:"sig,omitempty"`
}

func (r *WithdrawRequest) hash() common.Hash {
	cr := &WithdrawRequest{
		Address: r.Address,
		Amount: r.Amount,
	}
	b, err := json.Marshal(cr)
	if err != nil {
		return common.Hash{}
	}
	h := crypto.Keccak256Hash(b)
	return h
}

func SignWithdrawRequest (r *WithdrawRequest, priv *ecdsa.PrivateKey) error {
	// 1. check address
	signerAddr := crypto.PubkeyToAddress(priv.PublicKey)
	if signerAddr != common.HexToAddress(r.Address) {
		return fmt.Errorf("signer address not match")
	}
	// 2. hash
	h := r.hash()
	r.Hash = h.Hex()
	// 3. sign
	sig, err := crypto.Sign(h.Bytes(), priv)
	if err != nil {
		return err
	}
	r.Sig = hex.EncodeToString(sig)
	return nil
}

func (req *WithdrawRequest) VerifySignature() bool {
	sig, err := hex.DecodeString(req.Sig)
	if err != nil {
		return false
	}
	hash := common.HexToHash(req.Hash)
	// 1. recover pubkey
	pub, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		return false
	}
	// 2. verify v
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := new(big.Int).SetBytes([]byte{sig[64] + 27})
	vb := byte(v.Uint64() - 27)
	if !crypto.ValidateSignatureValues(vb, r, s, false) {
		return false
	}
	// 3. verify r,s
	cpub := crypto.CompressPubkey(pub)
	rs := sig[:64]
	if !crypto.VerifySignature(cpub, hash.Bytes(), rs) {
		return false
	}
	// 4. verify address
	recaddr := crypto.PubkeyToAddress(*pub)
	if recaddr != common.HexToAddress(req.Address) {
		return false
	}
	return true
}

func ValidateWithdraw(r *WithdrawRequest) error {
	AssetsLock.Lock()
	defer AssetsLock.Unlock()
	if common.HexToHash(r.Hash) != r.hash() {
		return fmt.Errorf("withdraw request hash error")
	}
	if r.VerifySignature() {
		user := common.HexToAddress(r.Address)
		// 判断user存在
		amount, ok := new(big.Int).SetString(r.Amount, 10)
		if !ok {
			return fmt.Errorf("withdraw amount error")
		}
		// 判断 ok & amount > 0
		// 获取余额
		userasset := GetUserAsset(user)
		// 选最长的一段
		today := time.Now().Add(-1 * time.Hour * 12).Round(time.Hour * 24).Unix()
		sendasset, _ := NewAsset(amount, uint64(today), 0)
		rem := sendasset.Sub(userasset)
		starttime := (*rem)[0].T
		endtime := starttime
		for i := 0; i < len(*rem); i++ {
			endtime = (*rem)[i].T
			if (*rem)[i].V.Cmp(big.NewInt(0)) >= 0 {
				continue
			} else {
				break
			}
		}
		if starttime == endtime {
			return fmt.Errorf("no enough balance")
		}
		// 构造Asset
		sendasset, _ = NewAsset(amount, starttime, endtime)
		// 构造WithdrawMsg
		msg := WithdrawMsg{
			Address:user,
			Asset:sendasset,
		}
		// 传入WithdrawCh
		WithdrawCh <- msg
	} else {
		return fmt.Errorf("verify signature failed")
	}
	return nil
}
