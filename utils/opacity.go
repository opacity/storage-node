package utils

func SliceStringChunks(s []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize

		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}

	return chunks
}
