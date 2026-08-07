package main

import (
	"strings"
	"sync"
	"testing"
)

func TestParseSenderSelection_accepts_supported_values(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want SenderSelection
	}{
		{name: "random", raw: "random", want: SenderSelectionRandom},
		{name: "round robin", raw: "round-robin", want: SenderSelectionRoundRobin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			raw := tt.raw

			// When
			got, err := parseSenderSelection(raw)

			// Then
			if err != nil {
				t.Fatalf("parseSenderSelection(%q): %v", raw, err)
			}
			if got != tt.want {
				t.Fatalf("parseSenderSelection(%q) = %q, want %q", raw, got, tt.want)
			}
		})
	}
}

func TestParseSenderSelection_rejects_unsupported_value(t *testing.T) {
	// Given
	raw := "first"

	// When
	_, err := parseSenderSelection(raw)

	// Then
	if err == nil {
		t.Fatal("parseSenderSelection returned nil error")
	}
	if !strings.Contains(err.Error(), "invalid --sender-selection") || !strings.Contains(err.Error(), raw) {
		t.Fatalf("parseSenderSelection error = %q, want flag name and invalid value", err)
	}
}

func TestSenderSelectionFlag_defaults_to_random_and_uses_prefixed_environment(t *testing.T) {
	// Given
	wantDefault := string(SenderSelectionRandom)
	wantEnvVar := "TX_OVERLOAD_SENDER_SELECTION"

	// When
	gotDefault := SenderSelectionFlag.Value
	gotEnvVar := SenderSelectionFlag.EnvVar

	// Then
	if gotDefault != wantDefault {
		t.Fatalf("SenderSelectionFlag.Value = %q, want %q", gotDefault, wantDefault)
	}
	if gotEnvVar != wantEnvVar {
		t.Fatalf("SenderSelectionFlag.EnvVar = %q, want %q", gotEnvVar, wantEnvVar)
	}
}

func TestSenderSelector_round_robin_wraps_deterministically(t *testing.T) {
	// Given
	selector := newSenderSelector(SenderSelectionRoundRobin)
	want := []int{0, 1, 2, 0, 1}
	got := make([]int, len(want))

	// When
	for i := range got {
		got[i] = selector.nextIndex(3)
	}

	// Then
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("selection %d = %d, want %d; all selections: %v", i, got[i], want[i], got)
		}
	}
}

func TestSenderSelector_round_robin_is_safe_for_concurrent_callers(t *testing.T) {
	// Given
	const (
		shardCount = 4
		callCount  = 64
	)
	selector := newSenderSelector(SenderSelectionRoundRobin)
	selected := make(chan int, callCount)
	var callers sync.WaitGroup
	callers.Add(callCount)

	// When
	for i := 0; i < callCount; i++ {
		go func() {
			defer callers.Done()
			selected <- selector.nextIndex(shardCount)
		}()
	}
	callers.Wait()
	close(selected)

	// Then
	counts := make([]int, shardCount)
	for index := range selected {
		counts[index]++
	}
	for index, count := range counts {
		if count != callCount/shardCount {
			t.Fatalf("index %d selected %d times, want %d; all counts: %v", index, count, callCount/shardCount, counts)
		}
	}
}
