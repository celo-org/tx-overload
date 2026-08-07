package main

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

func TestDistributorSend_routes_through_sender_selector(t *testing.T) {
	// Given
	shards := []Shard{
		{reqs: make(chan txmgr.TxCandidate, 1)},
		{reqs: make(chan txmgr.TxCandidate, 1)},
		{reqs: make(chan txmgr.TxCandidate, 1)},
	}
	distributor := &Distributor{
		m:        NewMetrics(),
		shards:   shards,
		selector: newSenderSelector(SenderSelectionRoundRobin),
	}

	// When
	for i := range shards {
		candidate := txmgr.TxCandidate{TxData: []byte{byte(i)}}
		if err := distributor.Send(context.Background(), candidate); err != nil {
			t.Fatalf("Send(%d): %v", i, err)
		}
	}

	// Then
	for i, shard := range shards {
		select {
		case candidate := <-shard.reqs:
			if len(candidate.TxData) != 1 || candidate.TxData[0] != byte(i) {
				t.Fatalf("shard %d received data %v, want [%d]", i, candidate.TxData, i)
			}
		default:
			t.Fatalf("shard %d received no candidate", i)
		}
	}
}
