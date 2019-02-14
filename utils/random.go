package utils

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

/*RandSeqFromRunes generates a random sequence from some runes*/
func RandSeqFromRunes(length int, sequence []rune) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = sequence[RandIndex(len(sequence))]
	}
	return string(b)
}

/*RandIndex returns a random index between 0 and the length of the slice passed in*/
func RandIndex(lengthOfSlice int) int {
	return rand.Intn(lengthOfSlice)
}
