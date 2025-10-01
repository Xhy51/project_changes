package project02

// DefaultStopwords returns a common English stopword set.
// No globals: caller injects or uses this helper.
func DefaultStopwords() map[string]struct{} {
	ws := []string{
		"a", "an", "the", "and", "or", "but",
		"to", "in", "of", "on", "for", "with", "as", "at", "by", "from",
		"is", "are", "was", "were", "be", "been", "being",
		"this", "that", "these", "those", "it", "its", "itself",
		"i", "me", "my", "myself", "we", "our", "ours", "ourselves",
		"you", "your", "yours", "yourself", "yourselves",
		"he", "him", "his", "himself", "she", "her", "hers", "herself",
		"they", "them", "their", "theirs", "themselves",
		"do", "does", "did", "doing",
		"have", "has", "had", "having",
		"not", "no", "nor", "only", "very", "too",
		"can", "could", "should", "would", "may", "might", "must", "will",
		"if", "then", "else", "than", "so", "because", "while", "when", "where",
		"about", "above", "below", "under", "over", "into", "out", "up", "down",
		"again", "further", "once", "here", "there",
	}
	m := make(map[string]struct{}, len(ws))
	for _, w := range ws {
		m[w] = struct{}{}
	}
	return m
}
