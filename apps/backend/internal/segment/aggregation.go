package segment

import (
	"context"
	"fmt"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"

	"github.com/fabula-studio/backend/internal/tree"
)

const (
	PromptVersion    = "unit-aggregator-v1"
	DefaultBatchSize = 8
	MaxUnitChars     = 12000
)

type UnitType string

const (
	UnitTypeScene   UnitType = "scene"
	UnitTypeSummary UnitType = "summary"
	UnitTypeDiscard UnitType = "discard"
)

type UnitResult struct {
	EndSentenceID  string   `json:"end_sentence_id"`
	UnitType       UnitType `json:"unit_type"`
	Summary        string   `json:"summary"`
	MainConflict   string   `json:"main_conflict"`
	Characters     []string `json:"characters"`
	Location       string   `json:"location"`
	TimeFrame      string   `json:"time_frame"`
	BoundaryReason string   `json:"boundary_reason"`
}

type TakeNextSentencesResponse struct {
	Sentences []Sentence     `json:"sentences"`
	Cursor    SentenceCursor `json:"cursor"`
}

type SentenceCursor struct {
	NextSentenceID string `json:"next_sentence_id,omitempty"`
	Remaining      int    `json:"remaining"`
	EndReached     bool   `json:"end_reached"`
}

type FinishUnitRequest struct {
	EndSentenceID  string   `json:"end_sentence_id"`
	UnitType       UnitType `json:"unit_type"`
	Summary        string   `json:"summary"`
	MainConflict   string   `json:"main_conflict"`
	Characters     []string `json:"characters"`
	Location       string   `json:"location"`
	TimeFrame      string   `json:"time_frame"`
	BoundaryReason string   `json:"boundary_reason"`
}

type aggregationStateKey struct{}

// AggregationState stores one forward-only aggregation run.
type AggregationState struct {
	Sentences []Sentence
	Start     int
	Cursor    int
	SeenEnd   int
	BatchSize int
	Result    *UnitResult
}

func NewAggregationState(sentences []Sentence, start, batchSize int) *AggregationState {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &AggregationState{
		Sentences: sentences,
		Start:     start,
		Cursor:    start,
		SeenEnd:   start - 1,
		BatchSize: batchSize,
	}
}

func WithAggregationState(ctx context.Context, state *AggregationState) context.Context {
	return context.WithValue(ctx, aggregationStateKey{}, state)
}

func AggregationStateFromContext(ctx context.Context) *AggregationState {
	state, _ := ctx.Value(aggregationStateKey{}).(*AggregationState)
	return state
}

func NewAggregationTools() []tool.Tool {
	return []tool.Tool{
		function.NewFunctionTool(takeNextSentences, function.WithName("take_next_sentences"), function.WithDescription("Returns the next fixed batch of complete sentences and advances the read cursor.")),
		function.NewFunctionTool(finishUnit, function.WithName("finish_unit"), function.WithDescription("Finishes the current story unit at a sentence ID with metadata and a concrete boundary reason.")),
	}
}

func takeNextSentences(ctx context.Context, _ map[string]interface{}) (TakeNextSentencesResponse, error) {
	state := AggregationStateFromContext(ctx)
	if state == nil {
		return TakeNextSentencesResponse{}, fmt.Errorf("no aggregation state in context")
	}
	if state.Cursor >= len(state.Sentences) {
		return TakeNextSentencesResponse{Cursor: SentenceCursor{Remaining: 0, EndReached: true}}, nil
	}
	end := state.Cursor + state.BatchSize
	if end > len(state.Sentences) {
		end = len(state.Sentences)
	}
	sentences := state.Sentences[state.Cursor:end]
	state.Cursor = end
	state.SeenEnd = end - 1
	cursor := SentenceCursor{
		Remaining:  len(state.Sentences) - state.Cursor,
		EndReached: state.Cursor >= len(state.Sentences),
	}
	if !cursor.EndReached {
		cursor.NextSentenceID = state.Sentences[state.Cursor].ID
	}
	return TakeNextSentencesResponse{Sentences: sentences, Cursor: cursor}, nil
}

func finishUnit(ctx context.Context, req FinishUnitRequest) (map[string]interface{}, error) {
	state := AggregationStateFromContext(ctx)
	if state == nil {
		return nil, fmt.Errorf("no aggregation state in context")
	}
	result := UnitResult(req)
	end, err := state.ValidateFinish(&result)
	if err != nil {
		return map[string]interface{}{"success": false, "error": err.Error()}, nil
	}
	state.Result = &result
	return map[string]interface{}{
		"success":                true,
		"end_sentence_id":        result.EndSentenceID,
		"next_start_sentence_id": nextSentenceID(state.Sentences, end+1),
		"unit_characters":        unitCharCount(state.Sentences[state.Start : end+1]),
	}, nil
}

func (s *AggregationState) ValidateFinish(result *UnitResult) (int, error) {
	if s == nil {
		return -1, fmt.Errorf("missing aggregation state")
	}
	if s.Start < 0 || s.Start >= len(s.Sentences) {
		return -1, fmt.Errorf("invalid start sentence index %d", s.Start)
	}
	end := sentenceIndexByID(s.Sentences, result.EndSentenceID)
	if end < 0 {
		return -1, fmt.Errorf("end_sentence_id %q was not found", result.EndSentenceID)
	}
	if end < s.Start {
		return -1, fmt.Errorf("end_sentence_id %q is before start sentence %q", result.EndSentenceID, s.Sentences[s.Start].ID)
	}
	if end > s.SeenEnd {
		return -1, fmt.Errorf("end_sentence_id %q has not been returned by take_next_sentences", result.EndSentenceID)
	}
	if result.UnitType != UnitTypeScene && result.UnitType != UnitTypeSummary && result.UnitType != UnitTypeDiscard {
		return -1, fmt.Errorf("invalid unit_type %q", result.UnitType)
	}
	if strings.TrimSpace(result.BoundaryReason) == "" {
		return -1, fmt.Errorf("boundary_reason is required")
	}
	if end < len(s.Sentences)-1 && isWeakBoundaryReason(result.BoundaryReason) {
		return -1, fmt.Errorf("boundary_reason is too weak for a non-final unit")
	}
	if unitCharCount(s.Sentences[s.Start:end+1]) > MaxUnitChars {
		return -1, fmt.Errorf("unit exceeds hard max character limit %d", MaxUnitChars)
	}
	return end, nil
}

func BuildTree(sentences []Sentence, units []UnitResult) (*tree.StoryTree, error) {
	st := tree.NewTree()
	root := &tree.StoryNode{ID: "node_000", ParentID: "", Level: -1, ChildrenIDs: make([]string, 0, len(units)), Decision: tree.DecisionKeep}
	st.AddNode(root)
	st.RootNodeID = root.ID

	start := 0
	for i, unit := range units {
		state := NewAggregationState(sentences, start, DefaultBatchSize)
		state.SeenEnd = len(sentences) - 1
		end, err := state.ValidateFinish(&unit)
		if err != nil {
			return nil, fmt.Errorf("unit %d validation failed: %w", i+1, err)
		}
		node := unitToNode(fmt.Sprintf("unit_%04d", i+1), root.ID, sentences[start:end+1], unit)
		st.AddNode(node)
		root.ChildrenIDs = append(root.ChildrenIDs, node.ID)
		start = end + 1
	}
	if start != len(sentences) {
		return nil, fmt.Errorf("units ended at sentence index %d, expected %d", start, len(sentences))
	}
	st.UpdateLeafIDs()
	return st, nil
}

func unitToNode(id, parentID string, sentences []Sentence, unit UnitResult) *tree.StoryNode {
	sourceIDs := make([]string, len(sentences))
	for i, sentence := range sentences {
		sourceIDs[i] = sentence.ID
	}
	return &tree.StoryNode{
		ID:                id,
		ParentID:          parentID,
		Level:             0,
		TextContent:       joinSentenceText(sentences),
		SourceChapter:     sentences[0].Chapter - 1,
		StartSentenceID:   sentences[0].ID,
		EndSentenceID:     sentences[len(sentences)-1].ID,
		SourceSentenceIDs: sourceIDs,
		BoundaryReason:    strings.TrimSpace(unit.BoundaryReason),
		UnitType:          string(unit.UnitType),
		Summary:           strings.TrimSpace(unit.Summary),
		MainConflict:      strings.TrimSpace(unit.MainConflict),
		Characters:        unit.Characters,
		Location:          strings.TrimSpace(unit.Location),
		TimeFrame:         strings.TrimSpace(unit.TimeFrame),
		IsComplete:        true,
		Decision:          decisionForUnitType(unit.UnitType),
	}
}

func decisionForUnitType(unitType UnitType) tree.NodeDecision {
	switch unitType {
	case UnitTypeSummary:
		return tree.DecisionSummarizeOnly
	case UnitTypeDiscard:
		return tree.DecisionDiscard
	default:
		return tree.DecisionKeep
	}
}

func joinSentenceText(sentences []Sentence) string {
	var builder strings.Builder
	for i, sentence := range sentences {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(sentence.Text)
	}
	return builder.String()
}

func sentenceIndexByID(sentences []Sentence, id string) int {
	for i, sentence := range sentences {
		if sentence.ID == id {
			return i
		}
	}
	return -1
}

func nextSentenceID(sentences []Sentence, index int) string {
	if index < 0 || index >= len(sentences) {
		return ""
	}
	return sentences[index].ID
}

func unitCharCount(sentences []Sentence) int {
	count := 0
	for _, sentence := range sentences {
		count += len(sentence.Text)
	}
	return count
}

func isWeakBoundaryReason(reason string) bool {
	trimmed := strings.TrimSpace(reason)
	weakReasons := []string{"内容较长", "差不多", "自然结束"}
	for _, weak := range weakReasons {
		if trimmed == weak {
			return true
		}
	}
	return false
}
