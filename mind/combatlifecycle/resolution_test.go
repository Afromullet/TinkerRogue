package combatlifecycle

import "testing"

func TestScale(t *testing.T) {
	tests := []struct {
		name     string
		input    Reward
		factor   float64
		expected Reward
	}{
		{
			name:     "full reward",
			input:    Reward{Gold: 100, Experience: 50, Mana: 10},
			factor:   1.0,
			expected: Reward{Gold: 100, Experience: 50, Mana: 10},
		},
		{
			name:     "half reward",
			input:    Reward{Gold: 100, Experience: 50, Mana: 10},
			factor:   0.5,
			expected: Reward{Gold: 50, Experience: 25, Mana: 5},
		},
		{
			name:     "double reward",
			input:    Reward{Gold: 100, Experience: 50, Mana: 10},
			factor:   2.0,
			expected: Reward{Gold: 200, Experience: 100, Mana: 20},
		},
		{
			name:     "zero factor",
			input:    Reward{Gold: 100, Experience: 50, Mana: 10},
			factor:   0.0,
			expected: Reward{Gold: 0, Experience: 0, Mana: 0},
		},
		{
			name:     "rounding up",
			input:    Reward{Gold: 3},
			factor:   0.5,
			expected: Reward{Gold: 2},
		},
		{
			name:     "zero reward scales to zero",
			input:    Reward{},
			factor:   1.5,
			expected: Reward{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Scale(tt.factor)
			if result != tt.expected {
				t.Errorf("Scale(%v) = %v, want %v", tt.factor, result, tt.expected)
			}
		})
	}
}

func TestRewardZeroFields(t *testing.T) {
	r := Reward{Gold: 100}
	if r.Experience != 0 || r.Mana != 0 {
		t.Error("zero-valued fields should be zero")
	}
}
