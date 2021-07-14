package utils

import (
	"encoding/base64"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func GenerateFileHandle() string {
	return RandHexString(64)
}

func GenerateMetadataV2Key() string {
	return HashStringV2(RandByteSlice(128))
}

func RandByteSlice(length int) []byte {
	b := make([]byte, length)
	rand.Read(b)

	return b
}

/*RandHexString generates a random hex string of the length passed in*/
func RandHexString(length int) string {
	return RandSeqFromRunes(length, []rune("abcdef01234567890"))
}

func RandB64String(byteLen int) string {
	b := make([]byte, byteLen)
	rand.Read(b)

	return base64.URLEncoding.EncodeToString(b)
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
