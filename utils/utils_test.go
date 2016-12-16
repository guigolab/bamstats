package utils

import "testing"

func TestMax(t *testing.T) {
	for i, c := range []struct {
		vals     [2]int
		expected int
	}{
		{[2]int{3, 7}, 7},
		{[2]int{5, 2}, 5},
	} {
		m := max(c.vals[0], c.vals[1])
		if m != c.expected {
			t.Errorf("[%d] Expected %v, got %v", i, c.expected, m)
		}
	}
}

func TestMin(t *testing.T) {
	for i, c := range []struct {
		vals     [2]int
		expected int
	}{
		{[2]int{3, 7}, 3},
		{[2]int{5, 2}, 2},
	} {
		m := min(c.vals[0], c.vals[1])
		if m != c.expected {
			t.Errorf("[%d] Expected %v, got %v", i, c.expected, m)
		}
	}
}
