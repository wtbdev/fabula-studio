package segment

import "fmt"

func testSentences(n int) []Sentence {
	sentences := make([]Sentence, n)
	for i := range sentences {
		sentences[i] = Sentence{ID: fmt.Sprintf("s_%06d", i+1), Chapter: 1, Index: i + 1, ChapterIndex: i + 1, Text: fmt.Sprintf("句子%d。", i+1)}
	}
	return sentences
}
