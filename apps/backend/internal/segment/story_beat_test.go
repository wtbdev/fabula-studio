package segment

import "testing"

func TestBuildSourceIndexPreservesSplitSentenceRanges(t *testing.T) {
	idx := BuildSourceIndex([]string{"第一句。第二句。", "第三句。"})
	if idx.Version != SourceIndexVersion {
		t.Fatalf("unexpected source index version %q", idx.Version)
	}
	if len(idx.Sentences) != 3 {
		t.Fatalf("expected 3 sentences, got %d", len(idx.Sentences))
	}
	text, err := idx.SentenceRangeText("s_000002", "s_000003")
	if err != nil {
		t.Fatalf("range text: %v", err)
	}
	if text != "第二句。\n第三句。" {
		t.Fatalf("unexpected range text %q", text)
	}
}

func TestValidateAndRepairStoryBeatsFillsGapsAndTrimsOverlap(t *testing.T) {
	idx := NewSourceIndex(testSentences(5))
	beats := ValidateAndRepairStoryBeats(idx, []StoryBeat{
		{StartSentenceID: "s_000002", EndSentenceID: "s_000004", Summary: "中段", DramaticPurpose: "推进", BoundaryReason: "行动结束"},
		{StartSentenceID: "s_000004", EndSentenceID: "s_000005", Summary: "尾声", BoundaryReason: "文本结束"},
	})
	if len(beats) != 3 {
		t.Fatalf("expected 3 repaired beats, got %d: %#v", len(beats), beats)
	}
	wantRanges := [][2]string{{"s_000001", "s_000001"}, {"s_000002", "s_000004"}, {"s_000005", "s_000005"}}
	for i, want := range wantRanges {
		if beats[i].ID == "" || beats[i].Sequence != i+1 {
			t.Fatalf("beat %d missing stable identity: %#v", i, beats[i])
		}
		if beats[i].StartSentenceID != want[0] || beats[i].EndSentenceID != want[1] {
			t.Fatalf("beat %d range = %s..%s, want %s..%s", i, beats[i].StartSentenceID, beats[i].EndSentenceID, want[0], want[1])
		}
	}
	if len(beats[1].SourceSentenceIDs) != 3 {
		t.Fatalf("middle beat source ids not repaired: %#v", beats[1].SourceSentenceIDs)
	}
}

func TestReconcileStoryBeatBoundariesMergesDuplicateAdjacentBeats(t *testing.T) {
	idx := NewSourceIndex(testSentences(2))
	beats := ReconcileStoryBeatBoundaries(idx, []StoryBeat{
		{StartSentenceID: "s_000001", EndSentenceID: "s_000001", Summary: "同一事件", Location: "地点", TimeFrame: "时间", Characters: []string{"甲"}, BoundaryReason: "继续"},
		{StartSentenceID: "s_000002", EndSentenceID: "s_000002", Summary: "同一事件", Location: "地点", TimeFrame: "时间", Characters: []string{"甲", "乙"}, BoundaryReason: "结束"},
	})
	if len(beats) != 1 {
		t.Fatalf("expected merged beat, got %#v", beats)
	}
	if beats[0].StartSentenceID != "s_000001" || beats[0].EndSentenceID != "s_000002" {
		t.Fatalf("unexpected merged range %#v", beats[0])
	}
	if len(beats[0].Characters) != 2 {
		t.Fatalf("characters not merged: %#v", beats[0].Characters)
	}
}
