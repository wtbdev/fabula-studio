package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/tool"

	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/util"
	fabulatool "github.com/fabula-studio/backend/internal/tool"
)

// GraphAnalyzerAgent updates the dynamic character graph based on a story node.
type GraphAnalyzerAgent struct {
	agent                           *llmagent.LLMAgent
	extractionAgent                 *llmagent.LLMAgent
	CustomExtractUpdateInstructions func(context.Context, string) (*graph.GraphUpdateResult, error)
}

const graphAnalyzerDesc = "基于故事节点更新角色状态、关系和故事冲突"
const graphAnalyzerToolPrompt = `你是一名故事连贯性专家。你维护一个动态的角色关系图谱。

你将收到：
1. 一个故事片段（当前节点）。
2. 当前的角色图谱状态（在该片段之前）。

你的任务是通过调用提供的工具来确定这个片段引入的变化：

1. **add_character**: 添加新引入的角色（id, name, current_goal, emotional_state, personality）
2. **update_character**: 更新已有角色的状态（id, current_goal, emotional_state, new_known_info）
3. **add_relation**: 添加或更新两角色之间的关系（from_id, to_id, type, description）
4. **add_conflict**: 添加新的未解决冲突（description, characters）
5. **resolve_conflict**: 标记已解决的冲突（index, resolution）

关键规则：
- 只报告本片段中出现的信息
- 不要推测未来的事件
- 如果角色学到了什么，用 new_known_info 调用 update_character
- 跟踪关系变化及其触发事件
- 可以依次调用多个工具以完整捕捉所有变化
- 完成后，输出你修改内容的简要总结`

const graphAnalyzerExtractionPrompt = `你是一名故事连贯性专家。你只负责从单个故事片段中提取动态图更新指令，不直接修改图谱。

输出必须是 JSON，结构如下：
{
  "new_characters": [{"id":"char_001","name":"...","current_goal":"...","emotional_state":"...","personality":["..."]}],
  "updated_characters": [{"id":"char_001","goal_change":"...","emotion_change":"...","new_known_info":"..."}],
  "relation_changes": [{"char_a":"char_001","char_b":"char_002","new_type":"...","new_description":"...","trigger_event":"..."}],
  "new_conflicts": [{"description":"...","involved":["char_001"],"status":"unresolved"}],
  "resolved_conflicts": ["已解决冲突的原描述"],
  "new_foreshadows": [{"description":"..."}]
}

关键规则：
- 只报告本片段中出现的信息，不推测未来事件
- 使用稳定角色 id；同名角色在不同片段中必须使用相同 id
- 如果不能确认角色已存在，也可以放入 new_characters；串行应用阶段会按 id 合并
- 只输出 JSON，不要输出解释文字`

// NewGraphAnalyzerAgent creates the graph analysis agent.
func NewGraphAnalyzerAgent(modelName, apiKey, baseURL string) *GraphAnalyzerAgent {
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

	// Combine validation tool with graph tools
	graphTools := fabulatool.NewGraphTools()
	allTools := make([]tool.Tool, 0, len(graphTools)+1)

	allTools = append(allTools, graphTools...)

	agt := llmagent.New("graph-analyzer",
		llmagent.WithModel(m),
		llmagent.WithDescription(graphAnalyzerDesc),
		llmagent.WithInstruction(graphAnalyzerToolPrompt),
		llmagent.WithGenerationConfig(genConfig),
		llmagent.WithTools(allTools),
		llmagent.WithMaxToolIterations(10),
	)
	extractionAgent := llmagent.New("graph-update-extractor",
		llmagent.WithModel(m),
		llmagent.WithDescription("从故事片段提取动态图更新指令"),
		llmagent.WithInstruction(graphAnalyzerExtractionPrompt),
		llmagent.WithGenerationConfig(genConfig),
	)
	return &GraphAnalyzerAgent{agent: agt, extractionAgent: extractionAgent}
}

// AnalyzeUpdate determines what changes a story node introduces to the graph.
// It uses tool calls to make incremental updates to the graph snapshot.
func (a *GraphAnalyzerAgent) AnalyzeUpdate(ctx context.Context, nodeText string, currentSnapshot *graph.GraphSnapshot) (*graph.GraphUpdateResult, error) {
	// Inject the snapshot into context so tools can access it
	snap := currentSnapshot.Clone()
	ctx = fabulatool.WithGraphSnapshot(ctx, snap)

	snapshotJSON, err := json.Marshal(currentSnapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	prompt := fmt.Sprintf("Current character graph state:\n```json\n%s\n```\n\nStory fragment:\n---\n%s\n---",
		string(snapshotJSON), nodeText)

	_, err = Run(ctx, a.agent, prompt)
	if err != nil {
		return nil, err
	}

	// Build the update result by comparing the modified snapshot with the original
	result := diffSnapshots(currentSnapshot, snap)

	// Log the changes for observability
	logChanges(result)

	return result, nil
}

// ExtractUpdateInstructions extracts graph update instructions without mutating a snapshot.
func (a *GraphAnalyzerAgent) ExtractUpdateInstructions(ctx context.Context, nodeText string) (*graph.GraphUpdateResult, error) {
	if a.CustomExtractUpdateInstructions != nil {
		return a.CustomExtractUpdateInstructions(ctx, nodeText)
	}
	prompt := fmt.Sprintf("Extract dynamic graph update instructions from this story fragment:\n---\n%s\n---", nodeText)
	raw, err := Run(ctx, a.extractionAgent, prompt)
	if err != nil {
		return nil, err
	}
	prepared, err := util.PrepareJSON(raw, "graph update instructions")
	if err != nil {
		return nil, err
	}
	var result graph.GraphUpdateResult
	if err := json.Unmarshal([]byte(prepared), &result); err != nil {
		return nil, fmt.Errorf("failed to parse graph update instructions: %w", err)
	}
	logChanges(&result)
	return &result, nil
}

// AnalyzeSceneUpdate updates the graph from the screenplay scene that was actually generated.
func (a *GraphAnalyzerAgent) AnalyzeSceneUpdate(ctx context.Context, generatedScene interface{}, currentSnapshot *graph.GraphSnapshot) (*graph.GraphUpdateResult, error) {
	sceneJSON, err := json.Marshal(generatedScene)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal generated scene for graph update: %w", err)
	}
	return a.AnalyzeUpdate(ctx, string(sceneJSON), currentSnapshot)
}

// diffSnapshots computes the difference between the original and modified snapshots.
func diffSnapshots(original, modified *graph.GraphSnapshot) *graph.GraphUpdateResult {
	result := &graph.GraphUpdateResult{}

	// Find new characters
	for id, cs := range modified.Characters {
		if _, exists := original.Characters[id]; !exists {
			result.NewCharacters = append(result.NewCharacters, *cs)
		}
	}

	// Find updated characters
	for id, modifiedCS := range modified.Characters {
		originalCS, exists := original.Characters[id]
		if !exists {
			continue
		}

		update := graph.CharacterUpdate{ID: id}
		hasChange := false

		if modifiedCS.CurrentGoal != originalCS.CurrentGoal {
			update.GoalChange = modifiedCS.CurrentGoal
			hasChange = true
		}
		if modifiedCS.EmotionalState != originalCS.EmotionalState {
			update.EmotionChange = modifiedCS.EmotionalState
			hasChange = true
		}
		// Check for new known_info entries
		if len(modifiedCS.KnownInfo) > len(originalCS.KnownInfo) {
			for _, info := range modifiedCS.KnownInfo[len(originalCS.KnownInfo):] {
				update.NewKnownInfo = info
				hasChange = true
			}
		}

		if hasChange {
			result.UpdatedCharacters = append(result.UpdatedCharacters, update)
		}
	}

	// Find new/changed relations
	originalRels := make(map[string]graph.Relation, len(original.Relations))
	for _, r := range original.Relations {
		originalRels[relationKey(r.CharA, r.CharB)] = r
	}

	for _, r := range modified.Relations {
		originalRel, ok := originalRels[relationKey(r.CharA, r.CharB)]
		if !ok || originalRel.Type != r.Type || originalRel.Description != r.Description {
			result.RelationChanges = append(result.RelationChanges, graph.RelationChange{
				CharA:          r.CharA,
				CharB:          r.CharB,
				NewType:        r.Type,
				NewDescription: r.Description,
			})
		}
	}

	// Find new conflicts
	if len(modified.UnresolvedConflicts) > len(original.UnresolvedConflicts) {
		for _, c := range modified.UnresolvedConflicts[len(original.UnresolvedConflicts):] {
			result.NewConflicts = append(result.NewConflicts, c)
		}
	}

	// Find resolved conflicts
	for i, mc := range modified.UnresolvedConflicts {
		if i < len(original.UnresolvedConflicts) {
			oc := original.UnresolvedConflicts[i]
			if oc.Status == "unresolved" && mc.Status == "resolved" {
				result.ResolvedConflicts = append(result.ResolvedConflicts, oc.Description)
			}
		}
	}

	return result
}

func relationKey(a, b string) string {
	if a > b {
		return b + ":" + a
	}
	return a + ":" + b
}

// logChanges logs the graph changes for observability.
func logChanges(result *graph.GraphUpdateResult) {
	if len(result.NewCharacters) > 0 {
		log.Printf("[GraphAnalyzer] Added %d characters", len(result.NewCharacters))
	}
	if len(result.UpdatedCharacters) > 0 {
		log.Printf("[GraphAnalyzer] Updated %d characters", len(result.UpdatedCharacters))
	}
	if len(result.RelationChanges) > 0 {
		log.Printf("[GraphAnalyzer] Changed %d relations", len(result.RelationChanges))
	}
	if len(result.NewConflicts) > 0 {
		log.Printf("[GraphAnalyzer] Added %d conflicts", len(result.NewConflicts))
	}
	if len(result.ResolvedConflicts) > 0 {
		log.Printf("[GraphAnalyzer] Resolved %d conflicts", len(result.ResolvedConflicts))
	}
}
