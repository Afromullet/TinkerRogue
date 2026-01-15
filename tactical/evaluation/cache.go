package evaluation

// DirtyCache provides lazy evaluation with round-based invalidation.
// Embed this in structs that need to cache expensive calculations
// and invalidate when game state changes.
type DirtyCache struct {
	lastUpdateRound int
	isDirty         bool
	isInitialized   bool
}

// NewDirtyCache creates a new cache in dirty state.
// First access will trigger computation.
func NewDirtyCache() *DirtyCache {
	return &DirtyCache{
		lastUpdateRound: -1,
		isDirty:         true,
		isInitialized:   false,
	}
}

// IsValid returns true if cached data is current for the given round.
// Returns false if cache is dirty or round has changed.
func (dc *DirtyCache) IsValid(currentRound int) bool {
	return dc.isInitialized && !dc.isDirty && dc.lastUpdateRound == currentRound
}

// MarkDirty invalidates the cache, forcing recomputation on next access.
// Call this when underlying data changes (e.g., squad moves, unit dies).
func (dc *DirtyCache) MarkDirty() {
	dc.isDirty = true
}

// MarkClean marks the cache as valid for the given round.
// Call this after successful recomputation.
func (dc *DirtyCache) MarkClean(currentRound int) {
	dc.isDirty = false
	dc.isInitialized = true
	dc.lastUpdateRound = currentRound
}

// IsDirty returns whether the cache needs recomputation.
func (dc *DirtyCache) IsDirty() bool {
	return dc.isDirty
}

// IsInitialized returns whether the cache has been computed at least once.
func (dc *DirtyCache) IsInitialized() bool {
	return dc.isInitialized
}

// GetLastUpdateRound returns the round number when cache was last updated.
func (dc *DirtyCache) GetLastUpdateRound() int {
	return dc.lastUpdateRound
}
