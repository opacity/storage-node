package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_RandSeqFromRunes(t *testing.T) {
	randSeq := RandSeqFromRunes(6, []rune("abcdefg01234567890"))
	assert.Equal(t, 6, len(randSeq))
}

func Test_RandIndex(t *testing.T) {
	for i := 0; i < 20; i++ {
		randIndex := RandIndex(5)
		assert.Equal(t, true, randIndex < 5 && randIndex >= 0)
	}
}
