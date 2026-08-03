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
