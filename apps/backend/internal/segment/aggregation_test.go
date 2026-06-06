package segment

import (
	"fmt"
	"testing"

	"github.com/fabula-studio/backend/internal/tree"
)

func testSentences(n int) []Sentence {
	sentences := make([]Sentence, n)
	for i := range sentences {
		sentences[i] = Sentence{ID: fmt.Sprintf("s_%06d", i+1), Chapter: 1, Index: i + 1, ChapterIndex: i + 1, Text: fmt.Sprintf("第%d句。", i+1)}
	}
	return sentences
}

func validUnit(endID string) UnitResult {
	return UnitResult{EndSentenceID: endID, UnitType: UnitTypeScene, Summary: "人物行动", MainConflict: "目标受阻", Characters: []string{"角色A"}, Location: "地点", TimeFrame: "时间", BoundaryReason: "下一句切换到新的行动目标"}
}

func TestValidateFinishRejectsUnseenEndSentence(t *testing.T) {
	state := NewAggregationState(testSentences(3), 0, 2)
	state.SeenEnd = 0
	if _, err := state.ValidateFinish(&UnitResult{EndSentenceID: "s_000002", UnitType: UnitTypeScene, BoundaryReason: "下一句切换地点"}); err == nil {
		t.Fatal("expected unseen sentence rejection")
	}
}

func TestValidateFinishRejectsEndBeforeStart(t *testing.T) {
	state := NewAggregationState(testSentences(3), 1, 2)
	state.SeenEnd = 2
	if _, err := state.ValidateFinish(&UnitResult{EndSentenceID: "s_000001", UnitType: UnitTypeScene, BoundaryReason: "下一句切换地点"}); err == nil {
		t.Fatal("expected end-before-start rejection")
	}
}

func TestValidateFinishAcceptsMidBatchBoundary(t *testing.T) {
	state := NewAggregationState(testSentences(4), 0, 4)
	state.SeenEnd = 3
	unit := validUnit("s_000002")
	end, err := state.ValidateFinish(&unit)
	if err != nil {
		t.Fatalf("expected valid boundary: %v", err)
	}
	if end != 1 {
		t.Fatalf("expected end index 1, got %d", end)
	}
}

func TestValidateFinishRejectsWeakReasonBeforeEnd(t *testing.T) {
	state := NewAggregationState(testSentences(3), 0, 3)
	state.SeenEnd = 2
	unit := validUnit("s_000001")
	unit.BoundaryReason = "自然结束"
	if _, err := state.ValidateFinish(&unit); err == nil {
		t.Fatal("expected weak boundary rejection")
	}
}

func TestBuildTreeMapsUnitsToDirectChildren(t *testing.T) {
	sentences := testSentences(3)
	st, err := BuildTree(sentences, []UnitResult{validUnit("s_000002"), {EndSentenceID: "s_000003", UnitType: UnitTypeSummary, Summary: "补充信息", BoundaryReason: "已到达文本结尾"}})
	if err != nil {
		t.Fatalf("build tree: %v", err)
	}
	root := st.GetNode("node_000")
	if root == nil || len(root.ChildrenIDs) != 2 {
		t.Fatalf("expected root with two children, got %#v", root)
	}
	first := st.GetNode("unit_0001")
	if first == nil {
		t.Fatal("missing unit_0001")
	}
	if first.ParentID != "node_000" || first.Level != 0 {
		t.Fatalf("unit not direct root child: %#v", first)
	}
	if first.StartSentenceID != "s_000001" || first.EndSentenceID != "s_000002" {
		t.Fatalf("wrong sentence range: %#v", first)
	}
	if first.TextContent != "第1句。\n第2句。" {
		t.Fatalf("unexpected text content: %q", first.TextContent)
	}
	if first.Decision != tree.DecisionKeep {
		t.Fatalf("scene should map to keep, got %s", first.Decision)
	}
	second := st.GetNode("unit_0002")
	if second.Decision != tree.DecisionSummarizeOnly {
		t.Fatalf("summary should map to summarize_only, got %s", second.Decision)
	}
}
