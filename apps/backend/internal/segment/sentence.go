package segment

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const SplitterVersion = "sentence-splitter-v1"

var noiseLines = map[string]struct{}{
	"展开全文阅读": {},
	"展开余文":   {},
}

// Sentence is a deterministic complete-sentence record used by unit aggregation.
type Sentence struct {
	ID           string `json:"id"`
	Chapter      int    `json:"chapter"`
	Index        int    `json:"index"`
	ChapterIndex int    `json:"chapter_index"`
	Text         string `json:"text"`
	Noise        bool   `json:"noise,omitempty"`
}

// SplitChapters converts chapter text into monotonic complete sentence records.
func SplitChapters(chapters []string) []Sentence {
	sentences := make([]Sentence, 0)
	globalIndex := 1
	for chapterIndex, chapter := range chapters {
		chapterSentenceIndex := 1
		for _, text := range splitChapterSentences(chapter) {
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				continue
			}
			_, isNoise := noiseLines[trimmed]
			if isNoise {
				continue
			}
			sentences = append(sentences, Sentence{
				ID:           fmt.Sprintf("s_%06d", globalIndex),
				Chapter:      chapterIndex + 1,
				Index:        globalIndex,
				ChapterIndex: chapterSentenceIndex,
				Text:         trimmed,
			})
			globalIndex++
			chapterSentenceIndex++
		}
	}
	return sentences
}

func splitChapterSentences(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isHeading(trimmed) {
			parts = append(parts, trimmed)
			continue
		}
		parts = append(parts, splitLineSentences(trimmed)...)
	}
	return parts
}

func isHeading(line string) bool {
	if strings.ContainsAny(line, "。！？!?；;") {
		return false
	}
	runeCount := utf8.RuneCountInString(line)
	if runeCount == 0 || runeCount > 24 {
		return false
	}
	for _, r := range line {
		if unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func splitLineSentences(line string) []string {
	var out []string
	start := 0
	for i, r := range line {
		if !isSentenceTerminator(r) {
			continue
		}
		end := i + utf8.RuneLen(r)
		end = consumeClosingMarks(line, end)
		appendTrimmed(&out, line[start:end])
		start = end
	}
	appendTrimmed(&out, line[start:])
	return out
}

func isSentenceTerminator(r rune) bool {
	switch r {
	case '。', '！', '？', '!', '?':
		return true
	default:
		return false
	}
}

func consumeClosingMarks(text string, pos int) int {
	for pos < len(text) {
		r, size := utf8.DecodeRuneInString(text[pos:])
		switch r {
		case '”', '’', '」', '』', '）', ')', '】', ']', '》', '〉', '"', '\'':
			pos += size
		default:
			return pos
		}
	}
	return pos
}

func appendTrimmed(out *[]string, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed != "" {
		*out = append(*out, trimmed)
	}
}
