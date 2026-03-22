package common

import (
	"testing"
)

func TestDirtyCache_NewCache(t *testing.T) {
	cache := NewDirtyCache()

	if !cache.IsDirty() {
		t.Error("New cache should be dirty")
	}
	if cache.IsInitialized() {
		t.Error("New cache should not be initialized")
	}
	if cache.GetLastUpdateRound() != -1 {
		t.Errorf("New cache lastUpdateRound = %d, want -1", cache.GetLastUpdateRound())
	}
}

func TestDirtyCache_IsValid(t *testing.T) {
	cache := NewDirtyCache()

	// New cache is not valid
	if cache.IsValid(0) {
		t.Error("New cache should not be valid")
	}

	// After marking clean, cache is valid for that round
	cache.MarkClean(5)
	if !cache.IsValid(5) {
		t.Error("Cache should be valid for round 5 after MarkClean(5)")
	}

	// Cache is not valid for different round
	if cache.IsValid(6) {
		t.Error("Cache should not be valid for round 6")
	}

	// After marking dirty, cache is not valid
	cache.MarkDirty()
	if cache.IsValid(5) {
		t.Error("Dirty cache should not be valid")
	}
}

func TestDirtyCache_MarkClean(t *testing.T) {
	cache := NewDirtyCache()

	cache.MarkClean(10)

	if cache.IsDirty() {
		t.Error("Cache should not be dirty after MarkClean")
	}
	if !cache.IsInitialized() {
		t.Error("Cache should be initialized after MarkClean")
	}
	if cache.GetLastUpdateRound() != 10 {
		t.Errorf("lastUpdateRound = %d, want 10", cache.GetLastUpdateRound())
	}
}

func TestDirtyCache_MarkDirty(t *testing.T) {
	cache := NewDirtyCache()
	cache.MarkClean(5)

	cache.MarkDirty()

	if !cache.IsDirty() {
		t.Error("Cache should be dirty after MarkDirty")
	}
	// Should still be initialized (data exists, just needs refresh)
	if !cache.IsInitialized() {
		t.Error("Cache should still be initialized after MarkDirty")
	}
}
