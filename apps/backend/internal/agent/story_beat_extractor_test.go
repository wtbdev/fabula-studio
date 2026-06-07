package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fabula-studio/backend/internal/segment"
)

func TestStoryBeatExtractorRunsWindowsConcurrentlyInDeterministicOrder(t *testing.T) {
	idx := segment.NewSourceIndex(testBeatSentences(48))
	var mu sync.Mutex
	active := 0
	maxActive := 0
	agt := &StoryBeatExtractorAgent{
		customExtractWindow: func(ctx context.Context, window segment.BeatWindow) ([]segment.StoryBeat, error) {
			mu.Lock()
			active++
			if active > maxActive {
				maxActive = active
			}
			mu.Unlock()

			time.Sleep(20 * time.Millisecond)

			mu.Lock()
			active--
			mu.Unlock()
			return []segment.StoryBeat{{
				StartSentenceID: window.StartSentenceID,
				EndSentenceID:   window.EndSentenceID,
				Summary:         fmt.Sprintf("window %d", window.Sequence),
				BoundaryReason:  "窗口结束",
			}}, nil
		},
	}

	beats, err := agt.ExtractWithConcurrency(context.Background(), idx, 3)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if maxActive < 2 {
		t.Fatalf("expected concurrent extraction, max active workers = %d", maxActive)
	}
	if len(beats) == 0 {
		t.Fatal("expected repaired beats")
	}
	for i, beat := range beats {
		if beat.Sequence != i+1 || beat.ID != fmt.Sprintf("beat_%03d", i+1) {
			t.Fatalf("beat %d lost deterministic identity: %#v", i, beat)
		}
		if i > 0 && beat.StartSentenceID <= beats[i-1].EndSentenceID {
			t.Fatalf("beats not reconciled in source order: prev=%#v current=%#v", beats[i-1], beat)
		}
	}
}

func TestStoryBeatExtractorConcurrencyFallbackKeepsCoverage(t *testing.T) {
	idx := segment.NewSourceIndex(testBeatSentences(30))
	agt := &StoryBeatExtractorAgent{
		customExtractWindow: func(ctx context.Context, window segment.BeatWindow) ([]segment.StoryBeat, error) {
			if window.Sequence == 2 {
				return nil, fmt.Errorf("boom")
			}
			return []segment.StoryBeat{{
				StartSentenceID: window.StartSentenceID,
				EndSentenceID:   window.EndSentenceID,
				Summary:         fmt.Sprintf("window %d", window.Sequence),
				BoundaryReason:  "窗口结束",
			}}, nil
		},
	}

	beats, err := agt.ExtractWithConcurrency(context.Background(), idx, 4)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(beats) == 0 {
		t.Fatal("expected beats")
	}
	if beats[0].StartSentenceID != "s_000001" || beats[len(beats)-1].EndSentenceID != "s_000030" {
		t.Fatalf("expected complete source coverage, got first=%#v last=%#v", beats[0], beats[len(beats)-1])
	}
}

func testBeatSentences(count int) []segment.Sentence {
	sentences := make([]segment.Sentence, count)
	for i := range sentences {
		sentences[i] = segment.Sentence{
			ID:           fmt.Sprintf("s_%06d", i+1),
			Chapter:      1,
			Index:        i + 1,
			ChapterIndex: i + 1,
			Text:         fmt.Sprintf("第 %d 句。", i+1),
		}
	}
	return sentences
}
