package segment

import "strings"

const MaxUnitChars = 3000

func sentenceIndexByID(sentences []Sentence, id string) int {
	for i, sentence := range sentences {
		if sentence.ID == id {
			return i
		}
	}
	return -1
}

func joinSentenceText(sentences []Sentence) string {
	parts := make([]string, 0, len(sentences))
	for _, sentence := range sentences {
		if text := strings.TrimSpace(sentence.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func unitCharCount(sentences []Sentence) int {
	count := 0
	for _, sentence := range sentences {
		count += len([]rune(sentence.Text))
	}
	return count
}
