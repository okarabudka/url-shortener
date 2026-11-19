package random

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRandomString(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{name: "size = 0", size: 0},
		{name: "size = 1", size: 1},
		{name: "size = 5", size: 5},
		{name: "size = 10", size: 10},
		{name: "size = 20", size: 20},
		{name: "size = 300", size: 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := NewRandomString(tt.size)

			assert.Len(t, str, tt.size)
			for _, r := range str {
				assert.True(t, strings.ContainsRune(alphabet, r), "unexpected character %q", r)
			}
		})
	}
}

// Two strings of length 8 collide with probability 62^-8 (about 5e-15), so a failure means the
// generator is broken, not unlucky. Shorter sizes are not compared: at size 1 a collision is 1 in 62.
func TestNewRandomString_Differs(t *testing.T) {
	assert.NotEqual(t, NewRandomString(8), NewRandomString(8))
}

func TestNewRandomString_UsesWholeAlphabet(t *testing.T) {
	seen := make(map[rune]bool)
	for _, r := range NewRandomString(10_000) {
		seen[r] = true
	}

	// Missing one of 62 characters in 10 000 draws has a probability far below 1e-60.
	assert.Len(t, seen, len(alphabet))
}
