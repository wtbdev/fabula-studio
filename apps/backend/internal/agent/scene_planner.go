package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"

	"github.com/fabula-studio/backend/internal/scene"

	"github.com/fabula-studio/backend/internal/util"
)

// ScenePlannerAgent decides how source-grounded scene candidates map to screenplay scenes.
type ScenePlannerAgent struct {
	agent                   *llmagent.LLMAgent
	CustomPlanFromCandidates func(context.Context, []scene.SceneCandidate) ([]*scene.ScenePlan, error)
}

const scenePlannerDesc = "规划故事候选节拍如何映射到剧本场景"
const scenePlannerPrompt = `你是一名剧本结构专家。你决定 source-grounded SceneCandidate 应该如何变成剧本场景。

你将收到 SceneCandidate 列表。每个 candidate 都有稳定 id、source_sentence_ids、summary、dramatic_purpose、conflict、location、time_frame、characters。

对每个候选节拍，决定它属于哪个剧本场景。一个 plan 永远对应一个最终场景，绝不能让一个 plan 代表多个生成场景；如果多个候选节拍发生在同一地点、时间、连续动作、同一戏剧目标内，可以合并进同一个 plan。
- purpose: 该场景在整个故事中的戏剧目的
- key_plot_points: 必须保留的关键情节点
- omit_details: 为节奏考虑可删减的小说细节

如果相邻候选节拍满足以下条件，可以合并为同一场景：
- 同一地点
- 同一时间
- 连续的动作
- 同一戏剧目标

输出一个场景规划 JSON 数组，每个包含：
- id: "plan_NNN"
- source_candidate_ids: 涉及的 candidate ID 数组；必须使用 candidate ID，不要输出 beat ID、sentence ID、source_node_ids 或 sourceNodeIds
- scene_count: 固定输出 1；即使输入要求多个场景也必须修正为 1
- purpose: 戏剧目的
- location: 主要地点
- time_frame: 主要时间
- characters: 出场角色名称/ID 数组
- key_plot_points: 必须保留的故事节拍
- omit_details: 可删减的细节
- sequence: 在完整剧本中的顺序（1, 2, 3...）

只输出合法 JSON，不要 markdown 代码块，不要额外注释。`

// NewScenePlannerAgent creates the scene planning agent.
func NewScenePlannerAgent(modelName, apiKey, baseURL string) *ScenePlannerAgent {
	opts := []openai.Option{}
	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelName, opts...)
	genConfig := model.GenerationConfig{
		Temperature: floatPtr(0.3),
	}

	agt := llmagent.New("scene-planner",
		llmagent.WithModel(m),
		llmagent.WithDescription(scenePlannerDesc),
		llmagent.WithInstruction(scenePlannerPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &ScenePlannerAgent{agent: agt}
}

func (a *ScenePlannerAgent) PlanFromCandidates(ctx context.Context, candidates []scene.SceneCandidate) ([]*scene.ScenePlan, error) {
	if a.CustomPlanFromCandidates != nil {
		return a.CustomPlanFromCandidates(ctx, candidates)
	}
	if len(candidates) == 0 {
	}
	candidatesJSON, err := json.Marshal(candidates)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scene candidates: %w", err)
	}
	prompt := fmt.Sprintf("Plan scenes for these source-grounded scene candidates. Use candidate IDs only in source_candidate_ids; do not emit source_node_ids/sourceNodeIds/sourceBeatIds. One plan writes exactly one scene, so every plan must have scene_count=1:\n```json\n%s\n```", string(candidatesJSON))

	plans, err := a.runPlanPrompt(ctx, prompt)
	if err != nil {
		return nil, err
	}
	return scene.ValidateAndRepairScenePlans(plans, candidates), nil
}

func (a *ScenePlannerAgent) runPlanPrompt(ctx context.Context, prompt string) ([]*scene.ScenePlan, error) {
	raw, err := Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	raw, err = util.PrepareJSON(raw, "scene planner output")
	if err != nil {
		return nil, err
	}
	raw = normalizeScenePlansJSON(raw)

	plans, err := parseScenePlansJSON(raw)
	if err != nil {
		return nil, err
	}
	return plans, nil
}

func parseScenePlansJSON(raw string) ([]*scene.ScenePlan, error) {
	var plans []*scene.ScenePlan
	if err := json.Unmarshal([]byte(raw), &plans); err == nil {
		return plans, nil
	}

	var wrapped struct {
		Plans  []*scene.ScenePlan `json:"plans"`
		Scenes []*scene.ScenePlan `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapped); err == nil {
		plans = append(plans, wrapped.Plans...)
		plans = append(plans, wrapped.Scenes...)
		if len(plans) > 0 {
			return plans, nil
		}
	}

	var planMap map[string]*scene.ScenePlan
	if err := json.Unmarshal([]byte(raw), &planMap); err != nil {
		return nil, fmt.Errorf("failed to parse scene plans JSON: %w\nraw: %s", err, raw)
	}
	plans = make([]*scene.ScenePlan, 0, len(planMap))
	for _, p := range planMap {
		plans = append(plans, p)
	}
	return plans, nil
}

// normalizeScenePlansJSON pre-processes LLM output to fix common JSON quirks.
func normalizeScenePlansJSON(raw string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return raw
	}
	normalizeValue(data)
	out, err := json.Marshal(data)
	if err != nil {
		return raw
	}
	return string(out)
}

func normalizeValue(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		normalizeScenePlanKeys(val)
		for _, v := range val {
			normalizeValue(v)
		}
	case []interface{}:
		for _, item := range val {
			normalizeValue(item)
		}
	}
}

func normalizeScenePlanKeys(val map[string]interface{}) {
	renameJSONKey(val, "sourceNodeIds", "source_candidate_ids")
	renameJSONKey(val, "sourceCandidateIds", "source_candidate_ids")
	renameJSONKey(val, "sourceBeatIds", "source_candidate_ids")
	renameJSONKey(val, "source_node_ids", "source_candidate_ids")
	renameJSONKey(val, "sceneCount", "scene_count")
	renameJSONKey(val, "timeFrame", "time_frame")
	renameJSONKey(val, "keyPlotPoints", "key_plot_points")
	renameJSONKey(val, "omitDetails", "omit_details")

	for _, key := range []string{"source_candidate_ids", "characters", "key_plot_points", "omit_details"} {
		if v, ok := val[key]; ok {
			val[key] = normalizeStringList(v)
		}
	}
}

func renameJSONKey(val map[string]interface{}, from, to string) {
	v, ok := val[from]
	if !ok {
		return
	}
	if _, exists := val[to]; !exists {
		val[to] = v
	}
	delete(val, from)
}

func normalizeStringList(v interface{}) []interface{} {
	switch val := v.(type) {
	case string:
		if val == "" {
			return []interface{}{}
		}
		return []interface{}{val}
	case []interface{}:
		out := make([]interface{}, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return []interface{}{}
	}
}
