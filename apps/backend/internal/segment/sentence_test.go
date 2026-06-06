package segment

import "testing"

func TestSplitChaptersChinesePunctuationAndQuotes(t *testing.T) {
	sentences := SplitChapters([]string{"她说：“你好！”我点点头。He ran away!"})
	want := []string{"她说：“你好！”", "我点点头。", "He ran away!"}
	if len(sentences) != len(want) {
		t.Fatalf("expected %d sentences, got %d: %#v", len(want), len(sentences), sentences)
	}
	for i, sentence := range sentences {
		if sentence.Text != want[i] {
			t.Fatalf("sentence %d: expected %q, got %q", i, want[i], sentence.Text)
		}
		wantID := "s_00000" + string(rune('1'+i))
		if sentence.ID != wantID {
			t.Fatalf("sentence %d: expected id %s, got %s", i, wantID, sentence.ID)
		}
		if sentence.Index != i+1 {
			t.Fatalf("sentence %d: expected index %d, got %d", i, i+1, sentence.Index)
		}
	}
}

func TestSplitChaptersHeadingsAndNoise(t *testing.T) {
	sentences := SplitChapters([]string{"草原\n展开全文阅读\n落日六号\n太阳落下。"})
	want := []string{"草原", "落日六号", "太阳落下。"}
	if len(sentences) != len(want) {
		t.Fatalf("expected %d sentences, got %d", len(want), len(sentences))
	}
	for i, sentence := range sentences {
		if sentence.Text != want[i] {
			t.Fatalf("sentence %d: expected %q, got %q", i, want[i], sentence.Text)
		}
		if sentence.Chapter != 1 || sentence.ChapterIndex != i+1 {
			t.Fatalf("sentence %d: wrong chapter metadata: %#v", i, sentence)
		}
	}
}
