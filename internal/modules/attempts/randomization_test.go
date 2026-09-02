package attempts

import (
	"reflect"
	"testing"
)

func TestShuffleStableUsesAttemptKey(t *testing.T) {
	first := []int{1, 2, 3, 4, 5, 6}
	second := append([]int(nil), first...)
	shuffleStable(first, "attempt-1:questions")
	shuffleStable(second, "attempt-1:questions")
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("stable shuffle differs: %v != %v", first, second)
	}
}

func TestShuffleStableDoesNotMutateEmptySlice(t *testing.T) {
	var values []int
	shuffleStable(values, "attempt-1:questions")
	if len(values) != 0 {
		t.Fatalf("expected empty slice, got %v", values)
	}
}
