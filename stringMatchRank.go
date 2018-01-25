func letterPairs(str string) []string {
	result := []string{}
	for i := 0; i < len(str)-1; i++ {
		result = append(result, str[i:i+2])
	}
	return result
}

func wordLetterPairs(str string) []string {
	result := []string{}
	words := regexp.MustCompile(`\s`).Split(str, -1)
	for _, w := range words {
		pairsInWord := letterPairs(w)
		for _, p := range pairsInWord {
			result = append(result, p)
		}
	}
	return result
}

func CompareStrings(str1, str2 string) float32 {
	pairs1 := wordLetterPairs(str1)
	pairs2 := wordLetterPairs(str2)
	var intersection int = 0
	union := len(pairs1) + len(pairs2)

	for _, pair1 := range pairs1 {
		for i, pair2 := range pairs2 {
			if pair1 == pair2 {
				intersection++
				pairs2 = append(pairs2[:i], pairs2[i+1:]...) // remove element
				break
			}
		}
	}
	return (float32(2) * float32(intersection)) / float32(union)
}

//
