package withdraw

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
	Hash      string `json:"hash,omitempty"`
	Address   string `json:"address"`
	Amount    string `json:"amount"`
	Timestamp string `json:"timestamp"`
	Sig       string `json:"sig,omitempty"`
}

func (r *WithdrawRequest) MakeHash() common.Hash {
	cr := &WithdrawRequest{
		Address: r.Address,
		Amount: r.Amount,
		Timestamp: fmt.Sprintf("%v", time.Now().Unix()),
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
	h := r.MakeHash()
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


