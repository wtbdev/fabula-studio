package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"

	"github.com/fabula-studio/backend/internal/segment"
	"github.com/fabula-studio/backend/internal/util"
)

// UnitAggregatorAgent detects stable story-unit boundaries from a sentence stream.
type UnitAggregatorAgent struct {
	agent *llmagent.LLMAgent
}

const unitAggregatorDesc = "按完整句子流聚合稳定剧情单元"
const unitAggregatorPrompt = `你是一名剧情单元边界检测器。

规则：
- 只能通过工具读取句子，不能要求用户提供文本，不能倒回。
- 每次从当前未分配句子开始判断一个剧情单元。
- 必须调用 take_next_sentences 读取下一批完整句子。
- 不确定时默认继续当前单元。
- 只有当后续句子明显进入新的场景、时间、地点、冲突阶段，或已经到达文本结尾时，才结束当前单元。
- 必须且只能调用 finish_unit 一次。
- finish_unit.end_sentence_id 必须是当前单元最后一个句子的 ID。
- boundary_reason 必须具体说明为什么边界在该句之后，例如时间/地点/行动目标/冲突阶段变化；不能只写“内容较长”“差不多”“自然结束”。
- unit_type 只能是 scene、summary、discard。
- summary、main_conflict、characters、location、time_frame 必须基于当前单元，不要引入前后文之外的信息。

完成工具调用后，输出与 finish_unit 参数一致的合法 JSON，不要 markdown，不要额外注释。`

// NewUnitAggregatorAgent creates the sentence-stream aggregation agent.
func NewUnitAggregatorAgent(modelName, apiKey, baseURL string) *UnitAggregatorAgent {
	opts := []openai.Option{}
	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelName, opts...)
	genConfig := model.GenerationConfig{Temperature: floatPtr(0.2)}
	agt := llmagent.New("unit-aggregator",
		llmagent.WithModel(m),
		llmagent.WithDescription(unitAggregatorDesc),
		llmagent.WithInstruction(unitAggregatorPrompt),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithTools(segment.NewAggregationTools()),
		llmagent.WithMaxToolIterations(32),
	)
	return &UnitAggregatorAgent{agent: agt}
}

// Aggregate runs one forward-only aggregation pass starting at state.Start.
func (a *UnitAggregatorAgent) Aggregate(ctx context.Context, state *segment.AggregationState) (*segment.UnitResult, error) {
	ctx = segment.WithAggregationState(ctx, state)
	raw, err := Run(ctx, a.agent, "开始新的剧情单元。调用工具读取句子，直到你能确定边界。")
	if err != nil {
		return nil, err
	}
	if state.Result != nil {
		return state.Result, nil
	}
	raw, err = util.PrepareJSON(raw, "unit aggregation output")
	if err != nil {
		return nil, err
	}
	var result segment.UnitResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("failed to parse unit aggregation JSON: %w\nraw: %s", err, raw)
	}
	if _, err := state.ValidateFinish(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
