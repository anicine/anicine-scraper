package shared

import "strings"

func TextSimpleSimilarity(text1, text2 string) float64 {
	// Split texts into words
	words1 := strings.Split(text1, "-")
	words2 := strings.Split(text2, "-")

	// Calculate intersection of words
	intersection := 0
	for _, word := range words1 {
		for _, w2 := range words2 {
			if word == w2 {
				intersection++
				break // Avoid counting duplicates
			}
		}
	}

	// Calculate union of unique words
	union := len(words1) + len(words2) - intersection

	// Handle edge cases (empty texts)
	if union == 0 {
		return 0
	}

	// Calculate similarity score (intersection / union)
	similarity := float64(intersection) / float64(union) * 100

	return similarity
}

func TextAdvancedSimilarity(text1, text2 string) float64 {
	if text1 == "" || text2 == "" {
		return 0
	}
	// Calculate the Levenshtein distance between the two texts.
	distance := levenshteinDistance(text1, text2)

	// Calculate the maximum possible length of the two texts.
	maxLength := len(text1)
	if len(text2) > maxLength {
		maxLength = len(text2)
	}

	// Calculate the similarity percentage.
	similarity := float64(maxLength-distance) / float64(maxLength) * 100

	return similarity
}

// levenshteinDistance calculates the Levenshtein distance between two strings.
func levenshteinDistance(s, t string) int {
	m := len(s)
	n := len(t)
	d := make([][]int, m+1)
	for i := range d {
		d[i] = make([]int, n+1)
	}
	for i := 0; i <= m; i++ {
		d[i][0] = i
	}
	for j := 0; j <= n; j++ {
		d[0][j] = j
	}
	for j := 1; j <= n; j++ {
		for i := 1; i <= m; i++ {
			cost := 0
			if s[i-1] != t[j-1] {
				cost = 1
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}

	return d[m][n]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
