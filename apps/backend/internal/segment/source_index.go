package segment

import (
	"fmt"
	"strings"
)

const SourceIndexVersion = "source-index-v1"

// SourceSentenceRef is a stable, compact source sentence pointer for downstream artifacts.
type SourceSentenceRef struct {
	ID           string `json:"id"`
	Chapter      int    `json:"chapter"`
	Index        int    `json:"index"`
	ChapterIndex int    `json:"chapter_index"`
}

// SourceIndex is the sentence-level index all adaptation artifacts point back to.
type SourceIndex struct {
	Version   string     `json:"version"`
	Sentences []Sentence `json:"sentences"`
}

// NewSourceIndex builds a source index from SplitChapters output.
func NewSourceIndex(sentences []Sentence) *SourceIndex {
	indexed := make([]Sentence, len(sentences))
	copy(indexed, sentences)
	return &SourceIndex{Version: SourceIndexVersion, Sentences: indexed}
}

// BuildSourceIndex splits chapters and returns the source index used by beat extraction.
func BuildSourceIndex(chapters []string) *SourceIndex {
	return NewSourceIndex(SplitChapters(chapters))
}

// SentenceRange returns the inclusive sentence range for two source sentence IDs.
func (idx *SourceIndex) SentenceRange(startID, endID string) ([]Sentence, error) {
	if idx == nil {
		return nil, fmt.Errorf("missing source index")
	}
	start := sentenceIndexByID(idx.Sentences, startID)
	if start < 0 {
		return nil, fmt.Errorf("start_sentence_id %q was not found", startID)
	}
	end := sentenceIndexByID(idx.Sentences, endID)
	if end < 0 {
		return nil, fmt.Errorf("end_sentence_id %q was not found", endID)
	}
	if end < start {
		return nil, fmt.Errorf("end_sentence_id %q is before start_sentence_id %q", endID, startID)
	}
	return idx.Sentences[start : end+1], nil
}

// SentenceRangeText returns newline-joined source text for an inclusive sentence range.
func (idx *SourceIndex) SentenceRangeText(startID, endID string) (string, error) {
	sentences, err := idx.SentenceRange(startID, endID)
	if err != nil {
		return "", err
	}
	return joinSentenceText(sentences), nil
}

// RefFor returns the compact reference for a source sentence ID.
func (idx *SourceIndex) RefFor(id string) (SourceSentenceRef, bool) {
	if idx == nil {
		return SourceSentenceRef{}, false
	}
	for _, sentence := range idx.Sentences {
		if sentence.ID == id {
			return SourceSentenceRef{ID: sentence.ID, Chapter: sentence.Chapter, Index: sentence.Index, ChapterIndex: sentence.ChapterIndex}, true
		}
	}
	return SourceSentenceRef{}, false
}

// TextForIDs returns newline-joined source text for a set of sentence IDs in source order.
func (idx *SourceIndex) TextForIDs(ids []string) string {
	if idx == nil || len(ids) == 0 {
		return ""
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	parts := make([]string, 0, len(ids))
	for _, sentence := range idx.Sentences {
		if _, ok := want[sentence.ID]; ok {
			parts = append(parts, sentence.Text)
		}
	}
	return strings.Join(parts, "\n")
}
