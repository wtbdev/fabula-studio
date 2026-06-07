package segment

import (
	"fmt"
	"strings"
)

const (
	StoryBeatVersion         = "story-beat-v1"
	DefaultBeatWindowSize    = 24
	DefaultBeatWindowOverlap = 4
	MaxStoryBeatChars        = MaxUnitChars
)

// BeatWindow is a bounded sentence window sent to the story-beat extractor.
type BeatWindow struct {
	ID              string     `json:"id"`
	Sequence        int        `json:"sequence"`
	StartSentenceID string     `json:"start_sentence_id"`
	EndSentenceID   string     `json:"end_sentence_id"`
	Sentences       []Sentence `json:"sentences"`
}

// StoryBeat is an adaptation-level narrative beat with stable source sentence bounds.
type StoryBeat struct {
	ID                string   `json:"id"`
	Sequence          int      `json:"sequence"`
	StartSentenceID   string   `json:"start_sentence_id"`
	EndSentenceID     string   `json:"end_sentence_id"`
	SourceSentenceIDs []string `json:"source_sentence_ids"`
	Summary           string   `json:"summary"`
	DramaticPurpose   string   `json:"dramatic_purpose"`
	Conflict          string   `json:"conflict"`
	Characters        []string `json:"characters"`
	Location          string   `json:"location"`
	TimeFrame         string   `json:"time_frame"`
	BoundaryReason    string   `json:"boundary_reason"`
}

// NewBeatWindows slices the source index into overlapping extraction windows.
func NewBeatWindows(idx *SourceIndex, size, overlap int) []BeatWindow {
	if idx == nil || len(idx.Sentences) == 0 {
		return nil
	}
	if size <= 0 {
		size = DefaultBeatWindowSize
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size - 1
	}
	step := size - overlap
	windows := make([]BeatWindow, 0, (len(idx.Sentences)+step-1)/step)
	for start, sequence := 0, 1; start < len(idx.Sentences); sequence++ {
		end := start + size
		if end > len(idx.Sentences) {
			end = len(idx.Sentences)
		}
		sentences := make([]Sentence, end-start)
		copy(sentences, idx.Sentences[start:end])
		windows = append(windows, BeatWindow{
			ID: fmt.Sprintf("beat_window_%03d", sequence), Sequence: sequence,
			StartSentenceID: sentences[0].ID, EndSentenceID: sentences[len(sentences)-1].ID,
			Sentences: sentences,
		})
		if end == len(idx.Sentences) {
			break
		}
		start += step
	}
	return windows
}

// ValidateAndRepairStoryBeats normalizes LLM beat output into a complete monotonic source coverage.
func ValidateAndRepairStoryBeats(idx *SourceIndex, beats []StoryBeat) []StoryBeat {
	if idx == nil || len(idx.Sentences) == 0 {
		return nil
	}
	valid := make([]StoryBeat, 0, len(beats))
	lastEnd := -1
	for _, beat := range beats {
		start := sentenceIndexByID(idx.Sentences, beat.StartSentenceID)
		end := sentenceIndexByID(idx.Sentences, beat.EndSentenceID)
		if start < 0 || end < start {
			continue
		}
		if start <= lastEnd {
			start = lastEnd + 1
			if start > end {
				continue
			}
		}
		if start > lastEnd+1 {
			valid = append(valid, fallbackBeat(idx.Sentences[lastEnd+1:start], len(valid)+1, "补齐未覆盖的源句范围"))
		}
		if unitCharCount(idx.Sentences[start:end+1]) > MaxStoryBeatChars {
			for _, split := range splitOversizeBeat(idx.Sentences[start:end+1], len(valid)+1, beat) {
				valid = append(valid, split)
			}
		} else {
			valid = append(valid, repairBeat(idx.Sentences[start:end+1], len(valid)+1, beat))
		}
		lastEnd = end
	}
	if lastEnd < len(idx.Sentences)-1 {
		valid = append(valid, fallbackBeat(idx.Sentences[lastEnd+1:], len(valid)+1, "补齐尾部源句范围"))
	}
	return valid
}

// ReconcileStoryBeatBoundaries deterministically merges duplicate/empty repaired beats.
func ReconcileStoryBeatBoundaries(idx *SourceIndex, beats []StoryBeat) []StoryBeat {
	if len(beats) == 0 {
		return ValidateAndRepairStoryBeats(idx, beats)
	}
	repaired := ValidateAndRepairStoryBeats(idx, beats)
	if len(repaired) < 2 {
		return repaired
	}
	merged := make([]StoryBeat, 0, len(repaired))
	for _, beat := range repaired {
		if len(merged) == 0 {
			merged = append(merged, beat)
			continue
		}
		prev := &merged[len(merged)-1]
		if shouldMergeBeats(*prev, beat) {
			mergeBeatInto(prev, beat)
			continue
		}
		beat.Sequence = len(merged) + 1
		beat.ID = fmt.Sprintf("beat_%03d", beat.Sequence)
		merged = append(merged, beat)
	}
	for i := range merged {
		merged[i].Sequence = i + 1
		merged[i].ID = fmt.Sprintf("beat_%03d", i+1)
	}
	return merged
}

func repairBeat(sentences []Sentence, sequence int, beat StoryBeat) StoryBeat {
	out := beat
	out.Sequence = sequence
	out.ID = fmt.Sprintf("beat_%03d", sequence)
	out.StartSentenceID = sentences[0].ID
	out.EndSentenceID = sentences[len(sentences)-1].ID
	out.SourceSentenceIDs = sentenceIDs(sentences)
	out.Summary = strings.TrimSpace(out.Summary)
	if out.Summary == "" {
		out.Summary = compactSummary(sentences)
	}
	out.DramaticPurpose = strings.TrimSpace(out.DramaticPurpose)
	if out.DramaticPurpose == "" {
		out.DramaticPurpose = "保留源文本中的连续行动/信息变化"
	}
	out.Conflict = strings.TrimSpace(out.Conflict)
	out.Location = strings.TrimSpace(out.Location)
	out.TimeFrame = strings.TrimSpace(out.TimeFrame)
	out.BoundaryReason = strings.TrimSpace(out.BoundaryReason)
	if out.BoundaryReason == "" {
		out.BoundaryReason = "源句范围连续且边界已校正"
	}
	out.Characters = compactStrings(out.Characters)
	return out
}

func fallbackBeat(sentences []Sentence, sequence int, reason string) StoryBeat {
	return repairBeat(sentences, sequence, StoryBeat{Summary: compactSummary(sentences), DramaticPurpose: "保留源文本中的连续行动/信息变化", BoundaryReason: reason})
}

func splitOversizeBeat(sentences []Sentence, startSequence int, beat StoryBeat) []StoryBeat {
	out := make([]StoryBeat, 0)
	for start := 0; start < len(sentences); {
		end := start + 1
		chars := len([]rune(sentences[start].Text))
		for end < len(sentences) {
			nextChars := chars + len([]rune(sentences[end].Text))
			if nextChars > MaxStoryBeatChars {
				break
			}
			chars = nextChars
			end++
		}
		part := beat
		part.BoundaryReason = strings.TrimSpace(part.BoundaryReason)
		if part.BoundaryReason == "" {
			part.BoundaryReason = "超长节拍按源句范围拆分"
		}
		out = append(out, repairBeat(sentences[start:end], startSequence+len(out), part))
		start = end
	}
	return out
}

func shouldMergeBeats(a, b StoryBeat) bool {
	return strings.TrimSpace(a.Summary) == strings.TrimSpace(b.Summary) && strings.TrimSpace(a.Location) == strings.TrimSpace(b.Location) && strings.TrimSpace(a.TimeFrame) == strings.TrimSpace(b.TimeFrame)
}

func mergeBeatInto(dst *StoryBeat, src StoryBeat) {
	dst.EndSentenceID = src.EndSentenceID
	dst.SourceSentenceIDs = append(dst.SourceSentenceIDs, src.SourceSentenceIDs...)
	if src.Conflict != "" && !strings.Contains(dst.Conflict, src.Conflict) {
		dst.Conflict = joinNonEmpty(dst.Conflict, src.Conflict)
	}
	dst.Characters = compactStrings(append(dst.Characters, src.Characters...))
	dst.BoundaryReason = joinNonEmpty(dst.BoundaryReason, src.BoundaryReason)
}

func sentenceIDs(sentences []Sentence) []string {
	ids := make([]string, len(sentences))
	for i, sentence := range sentences {
		ids[i] = sentence.ID
	}
	return ids
}

func compactSummary(sentences []Sentence) string {
	text := joinSentenceText(sentences)
	runes := []rune(text)
	if len(runes) <= 160 {
		return text
	}
	return string(runes[:157]) + "..."
}

func compactStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func joinNonEmpty(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" || strings.Contains(a, b) {
		return a
	}
	return a + "；" + b
}
