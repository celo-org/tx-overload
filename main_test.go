package main

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Expected values were verified against the chaos testnet (chain id 11162320,
// pragueTime 0) by binary-searching eth_call for the exact gas limit at which
// the node stops returning "intrinsic gas too low".
func TestIntrinsicGas(t *testing.T) {
	erc20Data := append(common.Hex2Bytes("a9059cbb"), append(
		common.LeftPadBytes(bytes.Repeat([]byte{0xab}, 20), 32),
		common.LeftPadBytes(nil, 32)...)...)

	tests := []struct {
		name string
		data []byte
		want uint64
	}{
		{"empty", nil, 21000},
		{"100 non-zero bytes", bytes.Repeat([]byte{0xab}, 100), 25000},
		{"500 non-zero bytes", bytes.Repeat([]byte{0xab}, 500), 41000},
		{"1000 non-zero bytes", bytes.Repeat([]byte{0xab}, 1000), 61000},
		{"1000 zero bytes", make([]byte, 1000), 31000},
		{"erc20 transfer", erc20Data, 22400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := intrinsicGas(tt.data)
			if err != nil {
				t.Fatalf("intrinsicGas: %v", err)
			}
			if got != tt.want {
				t.Errorf("intrinsicGas = %d, want %d", got, tt.want)
			}
		})
	}
}

// erc20 mode must target the configured token, not a hardcoded one, and must
// still encode a well-formed transfer(address,uint256) call.
func TestErc20CandidateUsesConfiguredToken(t *testing.T) {
	token := common.HexToAddress("0x471EcE3750Da237f93B8E339c536989b8978a438")
	tx := &TxOverload{TxMode: Erc20, TokenAddress: token}

	c, err := tx.generateErc20TxCandidate()
	if err != nil {
		t.Fatalf("generateErc20TxCandidate: %v", err)
	}
	if c.To == nil || *c.To != token {
		t.Errorf("To = %v, want %v", c.To, token)
	}
	if len(c.TxData) != 68 {
		t.Fatalf("TxData length = %d, want 68 (4 selector + 32 addr + 32 amount)", len(c.TxData))
	}
	if got := common.Bytes2Hex(c.TxData[:4]); got != "a9059cbb" {
		t.Errorf("selector = %s, want a9059cbb", got)
	}
	if c.GasLimit != 22400 {
		t.Errorf("GasLimit = %d, want 22400 (EIP-7623 floor for this payload)", c.GasLimit)
	}
}

// The floor must never price a payload below the legacy rule it supersedes.
func TestIntrinsicGasNeverBelowLegacy(t *testing.T) {
	for size := 0; size <= 2000; size += 250 {
		data := bytes.Repeat([]byte{0xab}, size)
		got, err := intrinsicGas(data)
		if err != nil {
			t.Fatalf("intrinsicGas(%d bytes): %v", size, err)
		}
		if legacy := 21000 + uint64(16*size); got < legacy {
			t.Errorf("%d bytes: intrinsicGas = %d, below legacy %d", size, got, legacy)
		}
	}
}
