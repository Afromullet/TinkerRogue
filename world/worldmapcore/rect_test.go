package worldmapcore

import "testing"

func TestNewRect(t *testing.T) {
	r := NewRect(2, 3, 4, 5)
	if r.X1 != 2 || r.Y1 != 3 || r.X2 != 6 || r.Y2 != 8 {
		t.Errorf("NewRect(2,3,4,5) = %+v, want X1=2 Y1=3 X2=6 Y2=8", r)
	}
}

func TestRectCenter(t *testing.T) {
	tests := []struct {
		name         string
		rect         Rect
		wantX, wantY int
	}{
		{"even extents", NewRect(2, 2, 6, 6), 5, 5},
		{"odd extents truncate", NewRect(0, 0, 5, 3), 2, 1},
		{"unit rect", NewRect(4, 7, 1, 1), 4, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := tt.rect.Center()
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("Center() = (%d, %d), want (%d, %d)", x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestRectIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b Rect
		want bool
	}{
		{"overlapping", NewRect(0, 0, 5, 5), NewRect(3, 3, 5, 5), true},
		{"fully disjoint", NewRect(0, 0, 3, 3), NewRect(10, 10, 3, 3), false},
		{"contained", NewRect(0, 0, 10, 10), NewRect(2, 2, 3, 3), true},
		// The next two pin the current inclusive-bounds behavior (see
		// WORLD_TECH_DEBT 1.10): edge-sharing and corner-touching rects
		// report as intersecting even though their carved interiors do not
		// overlap. Do NOT "fix" these to half-open here — callers must be
		// audited first.
		{"edge-sharing (inclusive bounds pinned)", NewRect(0, 0, 4, 4), NewRect(4, 0, 4, 4), true},
		{"corner-touching (inclusive bounds pinned)", NewRect(0, 0, 4, 4), NewRect(4, 4, 4, 4), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Intersect(tt.b); got != tt.want {
				t.Errorf("a.Intersect(b) = %v, want %v", got, tt.want)
			}
			if got := tt.b.Intersect(tt.a); got != tt.want {
				t.Errorf("b.Intersect(a) = %v, want %v (asymmetric result)", got, tt.want)
			}
		})
	}
}
