package main

import (
	"fmt"
	"math/rand"
	"sync/atomic"
)

type SenderSelection string

const (
	SenderSelectionRandom     SenderSelection = "random"
	SenderSelectionRoundRobin SenderSelection = "round-robin"
)

func parseSenderSelection(raw string) (SenderSelection, error) {
	switch SenderSelection(raw) {
	case SenderSelectionRandom:
		return SenderSelectionRandom, nil
	case SenderSelectionRoundRobin:
		return SenderSelectionRoundRobin, nil
	default:
		return "", fmt.Errorf("invalid --sender-selection value %q: must be random or round-robin", raw)
	}
}

type senderSelector struct {
	selection SenderSelection
	next      atomic.Uint64
}

func newSenderSelector(selection SenderSelection) *senderSelector {
	return &senderSelector{selection: selection}
}

func (s *senderSelector) nextIndex(shardCount int) int {
	switch s.selection {
	case SenderSelectionRandom:
		return rand.Intn(shardCount)
	case SenderSelectionRoundRobin:
		return int((s.next.Add(1) - 1) % uint64(shardCount))
	default:
		panic(fmt.Sprintf("unsupported sender selection %q", s.selection))
	}
}
